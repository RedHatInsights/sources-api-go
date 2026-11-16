package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

var (
	dryRun     = flag.Bool("dry-run", false, "Run without making changes")
	batchSize  = flag.Int("batch-size", 100, "Number of records to process per batch")
	workerPool = flag.Int("workers", 5, "Number of concurrent workers")
)

type reencryptionResult struct {
	authID  int64
	success bool
	err     error
}

func main() {
	flag.Parse()

	conf := config.Get()
	l.InitLogger(conf)
	util.InitializeEncryption()

	// Only works with database secret store
	if conf.SecretStore != config.DatabaseStore {
		l.Log.Fatal("Re-encryption only applies to database secret store")
	}

	l.Log.Infof("Starting re-encryption (dry-run: %v, batch-size: %d, workers: %d)",
		*dryRun, *batchSize, *workerPool)

	ctx := context.Background()

	// Initialize database connection
	dao.Init()

	database := dao.DB

	// Create re-encryption tracking table if not exists
	err := createTrackingTable(database)
	if err != nil {
		l.Log.Fatalf("Failed to create tracking table: %v", err)
	}

	// Get total count of authentications to re-encrypt
	var totalCount int64
	err = database.Model(&model.Authentication{}).
		Where("password IS NOT NULL").
		Where("id NOT IN (SELECT auth_id FROM reencryption_completed)").
		Count(&totalCount).Error
	if err != nil {
		l.Log.Fatalf("Failed to count authentications: %v", err)
	}

	l.Log.Infof("Total authentications to re-encrypt: %d", totalCount)

	if totalCount == 0 {
		l.Log.Info("No authentications to re-encrypt. Exiting.")
		return
	}

	// Process in batches
	offset := 0
	reencrypted := 0
	failed := 0
	startTime := time.Now()

	for {
		auths := []model.Authentication{}

		err = database.Where("password IS NOT NULL").
			Where("id NOT IN (SELECT auth_id FROM reencryption_completed)").
			Limit(*batchSize).
			Offset(offset).
			Find(&auths).Error
		if err != nil {
			l.Log.Errorf("Failed to fetch batch at offset %d: %v", offset, err)
			offset += *batchSize

			continue
		}

		if len(auths) == 0 {
			l.Log.Info("No more authentications to process")
			break
		}

		// Process batch with worker pool
		results := processBatchConcurrently(ctx, database, auths, *workerPool, *dryRun)

		for _, result := range results {
			if result.success {
				reencrypted++
			} else {
				failed++

				l.Log.Errorf("Failed to re-encrypt auth %d: %v", result.authID, result.err)
			}
		}

		offset += len(auths)

		// Progress reporting
		elapsed := time.Since(startTime)
		rate := float64(reencrypted) / elapsed.Seconds()
		l.Log.Infof("Progress: %d/%d (%.1f%%) | Failed: %d | Rate: %.1f/sec",
			reencrypted, totalCount, float64(reencrypted)/float64(totalCount)*100, failed, rate)
	}

	elapsed := time.Since(startTime)
	l.Log.Infof("Re-encryption complete! Successful: %d, Failed: %d, Duration: %v", reencrypted, failed, elapsed)

	if failed > 0 {
		os.Exit(1)
	}
}

func processBatchConcurrently(ctx context.Context, database *gorm.DB, auths []model.Authentication, workers int, dryRun bool) []reencryptionResult {
	jobs := make(chan model.Authentication, len(auths))
	results := make(chan reencryptionResult, len(auths))

	// Start worker pool
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for auth := range jobs {
				result := processAuthentication(ctx, database, &auth, dryRun)
				results <- result
			}
		}()
	}

	// Send jobs
	for _, auth := range auths {
		jobs <- auth
	}

	close(jobs)

	// Wait for workers to finish
	wg.Wait()
	close(results)

	// Collect results
	var allResults []reencryptionResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}

func processAuthentication(ctx context.Context, database *gorm.DB, auth *model.Authentication, dryRun bool) reencryptionResult {
	result := reencryptionResult{authID: auth.DbID}

	// Decrypt with current key setup (will use fallback if needed)
	decrypted, err := auth.GetPassword()
	if err != nil {
		result.err = fmt.Errorf("failed to decrypt: %w", err)
		return result
	}

	if decrypted == nil {
		// No password to re-encrypt
		result.success = true
		return result
	}

	// Re-encrypt with new key
	encrypted, err := util.Encrypt(*decrypted)
	if err != nil {
		result.err = fmt.Errorf("failed to encrypt: %w", err)
		return result
	}

	if dryRun {
		l.Log.Debugf("[DRY-RUN] Would re-encrypt auth %d", auth.DbID)
		result.success = true

		return result
	}

	// Update in database within a transaction
	tx := database.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update password
	auth.Password = &encrypted
	err = tx.Save(auth).Error

	if err != nil {
		tx.Rollback()
		result.err = fmt.Errorf("failed to save: %w", err)

		return result
	}

	// Mark as completed
	err = tx.Exec("INSERT INTO reencryption_completed (auth_id, completed_at) VALUES (?, ?)",
		auth.DbID, time.Now()).Error

	if err != nil {
		tx.Rollback()
		result.err = fmt.Errorf("failed to mark as completed: %w", err)

		return result
	}

	err = tx.Commit().Error
	if err != nil {
		result.err = fmt.Errorf("failed to commit transaction: %w", err)
		return result
	}

	result.success = true

	return result
}

func createTrackingTable(database *gorm.DB) error {
	return database.Exec(`
		CREATE TABLE IF NOT EXISTS reencryption_completed (
			auth_id BIGINT PRIMARY KEY,
			completed_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`).Error
}
