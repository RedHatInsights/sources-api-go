package logger

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type GormLogger struct {
	Logger                  *logrus.Logger
	SlowThreshold           time.Duration
	SkipErrorRecordNotFound bool
}

func (l *GormLogger) LogMode(gormLogger.LogLevel) gormLogger.Interface { return l }

// Trace runs a SQL query and logs how long it took as well as the sql executed.
// By default the log entry is debug, but if the SQL is very slow it will log as
// warn.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	duration := float64(elapsed.Nanoseconds()) / 1e6
	fileWithLineNum := utils.FileWithLineNum()

	entry := getEntryFromContext(ctx)

	sqlFields := logrus.Fields{
		"rows":     rows,
		"duration": duration,
		"filename": fileWithLineNum,
	}

	switch {
	case err != nil:
		if l.SkipErrorRecordNotFound && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}

		entry.WithFields(sqlFields).Warn(sql)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		entry.WithFields(sqlFields).Warn("SLOW SQL: " + sql)
	default:
		// Use Trace level for SQL logging so it's only shown when LOG_LEVEL=TRACE.
		// This prevents sensitive data from appearing in logs at DEBUG level and above.
		entry.WithFields(sqlFields).Trace(sql)
	}
}

// Functions implementing the rest of the gorm logger interface - not used
// unless you were to call `DB.Logger.Info(...)`, which we don't use. Everything
// else goes through Trace
func (l *GormLogger) Debug(ctx context.Context, logMessage string, data ...interface{}) {
	getEntryFromContext(ctx).Debugf(logMessage, data...)
}

func (l *GormLogger) Info(ctx context.Context, logMessage string, data ...interface{}) {
	getEntryFromContext(ctx).Infof(logMessage, data...)
}

func (l *GormLogger) Warn(ctx context.Context, logMessage string, data ...interface{}) {
	getEntryFromContext(ctx).Warnf(logMessage, data...)
}

func (l *GormLogger) Error(ctx context.Context, logMessage string, data ...interface{}) {
	getEntryFromContext(ctx).Errorf(logMessage, data...)
}

func getEntryFromContext(ctx context.Context) *logrus.Entry {
	if ctx != nil && ctx.Value(EchoLogger{}) != nil {
		if entry, ok := ctx.Value(EchoLogger{}).(*logrus.Entry); ok {
			return entry
		}
	}

	// falling back to a default logger if the asserion fails
	return Log.WithFields(logrus.Fields{})
}
