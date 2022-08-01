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

// Functions implementing the rest of the gorm logger interface - not used
// unless you were to call `DB.Logger.Info(...)`, which we don't use. Everything
// else goes through Trace
func (l *GormLogger) Debug(_ context.Context, logMessage string, data ...interface{}) {}
func (l *GormLogger) Info(_ context.Context, logMessage string, data ...interface{})  {}
func (l *GormLogger) Warn(_ context.Context, logMessage string, data ...interface{})  {}
func (l *GormLogger) Error(_ context.Context, logMessage string, data ...interface{}) {}

// Trace runs a SQL query and logs how long it took as well as the sql executed.
// By default the log entry is debug, but if the SQL is very slow it will log as
// warn.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	duration := float64(elapsed.Nanoseconds()) / 1e6
	fileWithLineNum := utils.FileWithLineNum()

	var entry *logrus.Entry
	var ok bool
	if ctx.Value(EchoLogger{}) != nil {
		if entry, ok = ctx.Value(EchoLogger{}).(*logrus.Entry); !ok {
			// falling back to a default logger if the asserion fails
			entry = Log.WithFields(logrus.Fields{})
		}
	} else {
		entry = Log.WithFields(logrus.Fields{})
	}

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
		entry.WithFields(sqlFields).Debug(sql)
	}
}
