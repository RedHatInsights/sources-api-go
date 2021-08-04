package logger

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	echoLog "github.com/labstack/gommon/log"
	"os"
	"time"

	appconf "github.com/RedHatInsights/sources-api-go/config"
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func forwardLogsToStderr(logHandler string) bool {
	return logHandler == "haberdasher"
}

func LogrusLogLevelFrom(configLogLevel string) logrus.Level {
	var logLevel logrus.Level

	switch configLogLevel {
	case "DEBUG":
		logLevel = logrus.DebugLevel
	case "ERROR":
		logLevel = logrus.ErrorLevel
	case "WARN":
		logLevel = logrus.WarnLevel
	default:
		logLevel = logrus.InfoLevel
	}

	if flag.Lookup("test.v") != nil {
		logLevel = logrus.FatalLevel
	}

	return logLevel
}

type CustomLoggerFormatter struct {
	Hostname              string
	AppName               string
	InjectedToOtherLogger bool
}

//Marshaler is an interface any type can implement to change its output in our production logs.
type Marshaler interface {
	MarshalLog() map[string]interface{}
}

func NewCustomLoggerFormatter(config *appconf.SourcesApiConfig, injectedToOtherLogger bool) *CustomLoggerFormatter {
	return &CustomLoggerFormatter{AppName: config.AppName, Hostname: config.Hostname, InjectedToOtherLogger: injectedToOtherLogger}
}

func basicLogFields(logLevel string, appName string, hostName string) map[string]interface{} {
	now := time.Now()

	return map[string]interface{}{
		"@timestamp": now.Format("2006-01-02T15:04:05.999Z"),
		"@version":   1,
		"level":      logLevel,
		"hostname":   hostName,
		"app":        appName,
		"labels":     map[string]interface{}{"app": appName},
		"tags":       []string{appName},
	}
}

func (f *CustomLoggerFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	data := basicLogFields(entry.Level.String(), f.AppName, f.Hostname)
	data["message"] = entry.Message

	if !f.InjectedToOtherLogger {
		var caller string

		if entry.Caller == nil {
			caller = ""
		} else {
			caller = entry.Caller.Func.Name()
		}

		data["caller"] = caller
	}

	if entry.Logger.Level == logrus.DebugLevel && entry.Caller != nil && !f.InjectedToOtherLogger {
		data["filename"] = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
	}

	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			data[k] = v.Error()
		case Marshaler:
			data[k] = v.MarshalLog()
		default:
			data[k] = v
		}
	}

	j, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	b.Write(j)

	b.Write([]byte("\n"))

	return b.Bytes(), nil
}


func LogOutputFrom(logHandler string) *os.File {
	var logOutput *os.File

	if forwardLogsToStderr(logHandler) {
		logOutput = os.Stderr
	} else {
		logOutput = os.Stdout
	}

	return logOutput
}

func InitLogger(config *appconf.SourcesApiConfig) *logrus.Logger {
	Log = &logrus.Logger{
		Out:          LogOutputFrom(config.LogHandler),
		Level:        LogrusLogLevelFrom(config.LogLevel),
		Formatter:    NewCustomLoggerFormatter(config, false),
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: true,
	}

	return Log
}
