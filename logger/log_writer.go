package logger

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

type LogWriter struct {
	Logger   *logrus.Logger
	Output   *os.File
	LogLevel string
}

func (lw *LogWriter) Write(data []byte) (n int, err error) {
	logFields := make(map[string]interface{})
	err = json.Unmarshal(data, &logFields)
	if err != nil {
		Log.Warn("Unmarshalling Error in LogWriter: " + err.Error())
		return 0, err
	}

	lw.logByLevel(logFields)
	return len(data), nil
}

func (lw *LogWriter) logByLevel(loggerFields map[string]interface{}) {
	logger := lw.Logger.WithFields(loggerFields)

	switch lw.LogLevel {
	case "DEBUG":
		logger.Info()
	case "ERROR":
		logger.Error()
	case "WARN":
		logger.Warn()
	default:
		logger.Info()
	}
}
