// Command reencrypt migrates credential ciphertext between encryption keys.
// It has two modes:
//
//	default (bootstrap): backfill the "password_hash_gcm" column from the
//	  legacy "password_hash" column — each row's AES-CBC ciphertext
//	  (pre-rotation key) is decrypted and re-encrypted with AES-GCM (active
//	  keyring key) into the new column, stamped with that key's id. Requires
//	  ENCRYPTION_KEY_OLD. This is the one-time CBC-to-GCM migration.
//
//	--rotate-keys: re-encrypt existing GCM rows with the active keyring key,
//	  for rotations after the first. Rows are selected by their stored key
//	  id differing from the active one; each is decrypted with the recorded
//	  key and re-encrypted with the active key. No schema change, no legacy
//	  key needed — this is the mode every future rotation uses.
//
// Both modes are non-destructive by construction: writes are compare-and-
// swap guarded so a concurrent API write always wins, and the source value
// is never overwritten in place by the bootstrap mode. See
// the encryption migration runbook for the full migration plan.
//
// Requires ENCRYPTION_KEYS (or the ENCRYPTION_KEY shorthand) to be set; the
// process refuses to start otherwise.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
)

var (
	dryRun     = flag.Bool("dry-run", false, "Decrypt and report without writing to the database")
	batchSize  = flag.Int("batch-size", 100, "Number of rows to fetch and write per batch (pacing knob)")
	rotateKeys = flag.Bool("rotate-keys", false, "Re-encrypt existing GCM rows with the active key (rotations after the first) instead of backfilling legacy CBC rows")
)

// maxConsecutiveBatchErrors is the number of consecutive batch fetch failures
// tolerated before the tool gives up and exits nonzero. A persistent database
// outage must terminate the run, not loop forever.
const maxConsecutiveBatchErrors = 3

// pendingAuth is the subset of the authentications table the tool cares
// about: the id, the legacy ciphertext, the GCM ciphertext, and the key id
// that encrypted it.
type pendingAuth struct {
	ID              int64   `gorm:"column:id"`
	PasswordHash    *string `gorm:"column:password_hash"`
	PasswordHashGcm *string `gorm:"column:password_hash_gcm"`
	EncryptionKeyID *string `gorm:"column:encryption_key_id"`
}

type rowStatus int

const (
	statusMigrated rowStatus = iota
	statusSkipped            // a concurrent API write won the race — the row is already current
	statusFailed
)

type rowResult struct {
	id     int64
	status rowStatus
	err    error
}

func main() {
	flag.Parse()

	conf := config.Get()
	l.InitLogger(conf)
	util.InitializeEncryption()

	if conf.SecretStore != config.DatabaseStore {
		l.Log.Fatal("Re-encryption only applies to the database secret store")
	}

	if !*rotateKeys && !util.LegacyKeyPresent() {
		l.Log.Fatal(`ENCRYPTION_KEY_OLD must be set to read the legacy "password_hash" column`)
	}

	mode := "bootstrap (legacy CBC -> GCM)"
	if *rotateKeys {
		mode = fmt.Sprintf("rotate keys -> active key %q", util.ActiveKeyID())
	}

	dao.Init()
	database := dao.DB

	remaining, err := countRemaining(database, *rotateKeys)
	if err != nil {
		l.Log.Fatalf("Failed to count rows awaiting migration: %v", err)
	}

	l.Log.Infof("Starting re-encryption, mode: %s (dry-run: %v, batch-size: %d), rows remaining: %d",
		mode, *dryRun, *batchSize, remaining)

	if remaining == 0 {
		l.Log.Info("Nothing to migrate. Exiting.")
		return
	}

	startTime := time.Now()

	var migrated, skipped, failed int

	lastID := int64(0)
	consecutiveErrors := 0

	for {
		batch, err := fetchBatch(database, lastID, *batchSize, *rotateKeys)
		if err != nil {
			consecutiveErrors++

			l.Log.Errorf("Failed to fetch batch after id %d (consecutive errors: %d/%d): %v",
				lastID, consecutiveErrors, maxConsecutiveBatchErrors, err)

			if consecutiveErrors >= maxConsecutiveBatchErrors {
				l.Log.Fatalf("Giving up after %d consecutive batch fetch errors — migrated: %d, skipped: %d, failed: %d",
					maxConsecutiveBatchErrors, migrated, skipped, failed)
			}

			// Retry the same keyset page: lastID is not advanced, so no rows
			// are silently skipped.
			continue
		}

		consecutiveErrors = 0

		if len(batch) == 0 {
			break
		}

		// Rows are processed serially and in id order: each is an
		// independent, CAS-guarded single-row update whose order does not
		// matter, and the tool's job is to pace itself against a live
		// database, not to race it. Batch size is the pacing knob.
		for _, row := range batch {
			result := migrateRow(database, row, *rotateKeys, *dryRun)

			switch result.status {
			case statusMigrated:
				migrated++
			case statusSkipped:
				skipped++
			case statusFailed:
				failed++

				l.Log.Errorf("Failed to migrate authentication %d: %v", result.id, result.err)
			}
		}

		lastID = batch[len(batch)-1].ID

		l.Log.Infof("Progress: processed through id %d | migrated: %d, skipped: %d, failed: %d | elapsed: %v",
			lastID, migrated, skipped, failed, time.Since(startTime).Round(time.Second))
	}

	if remaining, err = countRemaining(database, *rotateKeys); err != nil {
		l.Log.Errorf("Failed to recount rows awaiting migration: %v", err)
	}

	l.Log.Infof("Re-encryption complete: migrated: %d, skipped: %d, failed: %d, remaining: %d, duration: %v",
		migrated, skipped, failed, remaining, time.Since(startTime).Round(time.Second))

	if failed > 0 {
		os.Exit(1)
	}
}

// remainingPredicate returns the WHERE clause selecting rows awaiting
// migration in the given mode. It doubles as the progress meter and the
// contract criterion from the rollout plan.
func remainingPredicate(rotate bool) (string, []interface{}) {
	if rotate {
		// GCM rows not stamped with the active key. The NULL arm also heals
		// rows written before key stamping existed.
		return "password_hash_gcm IS NOT NULL AND (encryption_key_id IS NULL OR encryption_key_id != ?)", []interface{}{util.ActiveKeyID()}
	}

	return "password_hash_gcm IS NULL AND password_hash IS NOT NULL", nil
}

func countRemaining(db *gorm.DB, rotate bool) (int64, error) {
	var count int64

	predicate, args := remainingPredicate(rotate)

	err := db.Model(&model.Authentication{}).
		Where(predicate, args...).
		Count(&count).Error

	return count, err
}

// fetchBatch returns the next keyset page of rows awaiting migration. Keyset
// pagination (id > lastID, ORDER BY id) instead of OFFSET: the result set
// shrinks as the tool progresses, which would make advancing offsets skip
// rows.
func fetchBatch(db *gorm.DB, afterID int64, limit int, rotate bool) ([]pendingAuth, error) {
	batch := []pendingAuth{}

	predicate, args := remainingPredicate(rotate)

	err := db.Model(&model.Authentication{}).
		Select("id, password_hash, password_hash_gcm, encryption_key_id").
		Where(predicate, args...).
		Where("id > ?", afterID).
		Order("id ASC").
		Limit(limit).
		Find(&batch).Error

	return batch, err
}

// migrateRow moves one row onto the active key. In bootstrap mode that means
// legacy-CBC-decrypt -> GCM-encrypt into the new columns; in rotate mode it
// means decrypt with the recorded key id -> re-encrypt with the active key.
// Writes are compare-and-swap guarded: if a concurrent API write changed the
// row between our fetch and our write, the update affects zero rows and the
// row is reported as skipped — the API write always wins.
func migrateRow(db *gorm.DB, row pendingAuth, rotate bool, dryRun bool) rowResult {
	if rotate {
		return rotateRow(db, row, dryRun)
	}

	return bootstrapRow(db, row, dryRun)
}

// bootstrapRow backfills one legacy CBC row into the GCM column, stamping
// the active key id. The legacy column is never written.
func bootstrapRow(db *gorm.DB, row pendingAuth, dryRun bool) rowResult {
	if row.PasswordHash == nil {
		// Cannot happen — the query filters on NOT NULL — but never trust a
		// nil password pointer.
		return rowResult{id: row.ID, status: statusFailed, err: fmt.Errorf("row has no legacy password")}
	}

	encrypted, err := transformLegacyCiphertext(*row.PasswordHash)
	if err != nil {
		return rowResult{id: row.ID, status: statusFailed, err: err}
	}

	if dryRun {
		return rowResult{id: row.ID, status: statusMigrated}
	}

	result := db.Model(&model.Authentication{}).
		Where("id = ? AND password_hash_gcm IS NULL", row.ID).
		Updates(map[string]interface{}{
			"password_hash_gcm": encrypted,
			"encryption_key_id": util.ActiveKeyID(),
		})

	if result.Error != nil {
		return rowResult{id: row.ID, status: statusFailed, err: fmt.Errorf("update: %w", result.Error)}
	}

	if result.RowsAffected == 0 {
		return rowResult{id: row.ID, status: statusSkipped}
	}

	return rowResult{id: row.ID, status: statusMigrated}
}

// rotateRow re-encrypts one GCM row with the active key. A NULL stored key
// id means the row predates key stamping and was encrypted with whatever
// key was active at the time — the active key is the only consistent
// assumption, and a wrong guess fails loudly (GCM authentication) rather
// than corrupting.
func rotateRow(db *gorm.DB, row pendingAuth, dryRun bool) rowResult {
	if row.PasswordHashGcm == nil {
		return rowResult{id: row.ID, status: statusFailed, err: fmt.Errorf("row has no GCM password")}
	}

	encrypted, err := rotateGCMCiphertext(*row.PasswordHashGcm, row.EncryptionKeyID)
	if err != nil {
		return rowResult{id: row.ID, status: statusFailed, err: err}
	}

	if dryRun {
		return rowResult{id: row.ID, status: statusMigrated}
	}

	// CAS on the exact key id value read (NULL-safe): a concurrent write
	// that already re-stamped the row makes this match zero rows.
	result := db.Model(&model.Authentication{}).
		Where("id = ? AND encryption_key_id IS NOT DISTINCT FROM ?", row.ID, row.EncryptionKeyID).
		Updates(map[string]interface{}{
			"password_hash_gcm": encrypted,
			"encryption_key_id": util.ActiveKeyID(),
		})

	if result.Error != nil {
		return rowResult{id: row.ID, status: statusFailed, err: fmt.Errorf("update: %w", result.Error)}
	}

	if result.RowsAffected == 0 {
		return rowResult{id: row.ID, status: statusSkipped}
	}

	return rowResult{id: row.ID, status: statusMigrated}
}

// transformLegacyCiphertext decrypts a legacy-column value with the
// pre-rotation key and re-encrypts the plaintext with AES-GCM and the
// active keyring key.
func transformLegacyCiphertext(legacy string) (string, error) {
	plaintext, err := util.DecryptLegacyCBC(legacy)
	if err != nil {
		return "", fmt.Errorf("legacy decrypt: %w", err)
	}

	encrypted, err := util.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("GCM encrypt: %w", err)
	}

	return encrypted, nil
}

// rotateGCMCiphertext decrypts a GCM value with the keyring entry named by
// recordedID (the active key when nil) and re-encrypts it with the active
// key.
func rotateGCMCiphertext(ciphertext string, recordedID *string) (string, error) {
	plaintext, err := decryptGCMByRecordedID(ciphertext, recordedID)
	if err != nil {
		return "", err
	}

	encrypted, err := util.Encrypt(plaintext)
	if err != nil {
		return "", fmt.Errorf("GCM encrypt: %w", err)
	}

	return encrypted, nil
}

func decryptGCMByRecordedID(ciphertext string, recordedID *string) (string, error) {
	if recordedID == nil {
		plaintext, err := util.DecryptGCM(ciphertext)
		if err != nil {
			return "", fmt.Errorf("decrypt with active key (row has no key id): %w", err)
		}

		return plaintext, nil
	}

	plaintext, err := util.DecryptGCMWithKeyID(ciphertext, *recordedID)
	if err != nil {
		return "", fmt.Errorf("decrypt with key %q: %w", *recordedID, err)
	}

	return plaintext, nil
}
