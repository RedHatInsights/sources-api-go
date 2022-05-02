package logger

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type CustomGORMLogger struct {
	Logger                  *logrus.Logger
	SlowThreshold           time.Duration
	SkipErrorRecordNotFound bool
	LogLevelForSqlLogs      string
}

func (l *CustomGORMLogger) LogMode(gormLogger.LogLevel) gormLogger.Interface {
	return l
}

func (l *CustomGORMLogger) Info(_ context.Context, logMessage string, data ...interface{}) {
	l.Logger.Info(logMessage, data)
}

func (l *CustomGORMLogger) Warn(_ context.Context, logMessage string, data ...interface{}) {
	l.Logger.Warn(logMessage, data)
}

func (l *CustomGORMLogger) Error(_ context.Context, logMessage string, data ...interface{}) {
	l.Logger.Error(logMessage, data)
}

func (l *CustomGORMLogger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	var ErrorRecordNotFound = errors.New("record not found")

	elapsed := time.Since(begin)
	sql, rows := fc()
	duration := float64(elapsed.Nanoseconds()) / 1e6
	fileWithLineNum := utils.FileWithLineNum()

	loggerEntry := l.Logger.WithFields(logrus.Fields{
		"rows":     rows,
		"duration": duration,
		"filename": fileWithLineNum,
		"log_type": SQLType,
	})

	switch {
	case err != nil:
		if errors.Is(err, ErrorRecordNotFound) && l.SkipErrorRecordNotFound {
			return
		}

		loggerEntry.Warn(sql)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		loggerEntry.Warn("SLOW SQL: " + sql)
	default:
		l.logByLevelWithFields(loggerEntry, sql)
	}
}

func (l *CustomGORMLogger) logByLevelWithFields(loggerWithFields *logrus.Entry, sql string) {
	switch l.LogLevelForSqlLogs {
	case "DEBUG":
		loggerWithFields.Debug(sql)
	case "ERROR":
		loggerWithFields.Error(sql)
	case "WARN":
		loggerWithFields.Warn(sql)
	default:
		loggerWithFields.Info(sql)
	}
}
