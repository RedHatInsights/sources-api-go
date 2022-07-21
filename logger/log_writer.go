package logger

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type LogWriter struct {
	Logger *logrus.Logger
}

func (lw *LogWriter) Write(data []byte) (n int, err error) {
	logFields := make(map[string]interface{})
	err = json.Unmarshal(data, &logFields)
	if err != nil {
		lw.Logger.Warn("Unmarshalling Error in LogWriter: " + err.Error())
		return 0, err
	}

	lw.Logger.WithFields(logFields).Info()
	return len(data), nil
}
