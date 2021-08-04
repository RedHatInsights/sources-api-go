package dao

import (
	"fmt"
	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/sirupsen/logrus"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var conf = config.Get()

func Init() {
	logger := &logrus.Logger{
		Out:       logging.LogOutputFrom(conf.LogHandler),
		Level:     logging.LogrusLogLevelFrom(conf.LogLevel),
		Formatter: logging.NewCustomLoggerFormatter(conf, true),
	}

	l := &logging.CustomGORMLogger{SkipErrorRecordNotFound: true,
		Logger:             logger,
		SlowThreshold:      time.Duration(conf.SlowSQLThreshold) * time.Second,
		LogLevelForSqlLogs: conf.LogLevelForSqlLogs,
	}

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
