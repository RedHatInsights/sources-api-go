package dao

import (
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	vault "github.com/hashicorp/vault/api"
	_ "github.com/jackc/pgx/v4/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB *gorm.DB

	vaultClient *vault.Client
	Vault       *vault.Logical

	conf = config.Get()
)

func Init() {
	l := &logging.CustomGORMLogger{
		SkipErrorRecordNotFound: true,
		Logger:                  logging.Log,
		SlowThreshold:           time.Duration(conf.SlowSQLThreshold) * time.Second,
		LogLevelForSqlLogs:      conf.LogLevelForSqlLogs,
	}

	// Open up the conn to postgres
	db, err := gorm.Open(postgres.Open(dbString()), &gorm.Config{Logger: l})
	if err != nil {
		panic(err)
	}
	DB = db
	rawDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	rawDB.SetMaxOpenConns(20)

	// Open up the conn to Vault
	cfg := vault.DefaultConfig()
	if cfg == nil {
		panic("Failed to parse Vault Config")
	}
	err = cfg.ReadEnvironment()
	if err != nil {
		panic(fmt.Sprintf("Failed to read Vault Environment: %v", err))
	}

	vaultClient, err = vault.NewClient(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to Create Vault Client: %v", err))
	}
	Vault = vaultClient.Logical()
}

func dbString() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=disable",
		config.Get().DatabaseUser,
		config.Get().DatabasePassword,
		config.Get().DatabaseName,
		config.Get().DatabaseHost,
		config.Get().DatabasePort,
	)
}
