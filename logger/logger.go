package logger

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	appconf "github.com/RedHatInsights/sources-api-go/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/labstack/echo/v4"
	echoLog "github.com/labstack/gommon/log"
	logrusEcho "github.com/neko-neko/echo-logrus/v2/log"
	lc "github.com/redhatinsights/platform-go-middlewares/logging/cloudwatch"
	"github.com/sirupsen/logrus"
)

const (
	DefaultType = "default"
	RequestType = "request"
	EchoType    = "echo"
	SQLType     = "sql"
)

var whiteListForFunctionsInBacktrace = []string{"RedHatInsights", "redhatinsights", "main"}
var Log *logrus.Logger

var cloudWatchHook *lc.Hook

// GetCloudWatchHook is an overrideable function which returns a CloudWatch Hook. The idea is to abstract away how the
// hook is grabbed, for easier testing in the future.
var GetCloudWatchHook func(config *appconf.SourcesApiConfig) *lc.Hook

// init sets the default function which provides a ClowdWatch Hook.
func init() {
	GetCloudWatchHook = singleCloudWatchLogrusHook
}

func singleCloudWatchLogrusHook(config *appconf.SourcesApiConfig) *lc.Hook {
	if cloudWatchHook == nil {
		cloudWatchHook = cloudWatchLogrusHook(config)
	}

	return cloudWatchHook
}

func AddHooksTo(logger *logrus.Logger, config *appconf.SourcesApiConfig) {
	hook := GetCloudWatchHook(config)
	if hook == nil {
		Log.Warn("Key or Secret are missing for logging to Cloud Watch.")
	} else {
		logger.AddHook(hook)
	}
}

func cloudWatchLogrusHook(config *appconf.SourcesApiConfig) *lc.Hook {
	key := config.AwsAccessKeyID
	secret := config.AwsSecretAccessKey
	region := config.AwsRegion
	group := config.LogGroup
	stream := config.Hostname

	if key != "" && secret != "" {
		cred := credentials.NewStaticCredentials(key, secret, "")
		awsconf := aws.NewConfig().WithRegion(region).WithCredentials(cred)
		hook, err := lc.NewBatchingHook(group, stream, awsconf, 10*time.Second)
		if err != nil {
			Log.Info(err)
		}

		return hook
	}

	return nil
}

func forwardLogsToStderr(logHandler string) bool {
	return logHandler == "haberdasher"
}

func LogrusLogLevelFrom(configLogLevel string) logrus.Level {
	var logLevel logrus.Level

	switch strings.ToUpper(configLogLevel) {
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

// StringContainAnySubString - TODO: Move this function to util package
func stringContainAnySubString(stringToSearch string, subStrings []string) bool {
	for _, subString := range subStrings {
		if strings.Contains(stringToSearch, subString) {
			return true
		}
	}
	return false
}

/*
  example:
	functionNameFromPath("github.com/RedHatInsights/sources-api-go/middleware.Timing.func1")
	=> "middleware.Timing.func1"
*/
func functionNameFromPath(functionWithPath string) string {
	functionPathParts := strings.Split(functionWithPath, "/")
	partsLength := len(functionPathParts)
	if partsLength == 0 {
		return ""
	}

	return functionPathParts[partsLength-1]
}

func backtrace() []string {
	var filenames []string

	pc := make([]uintptr, 1000)
	n := runtime.Callers(0, pc)
	frames := runtime.CallersFrames(pc[:n])

	for frame, isNextFrameValid := frames.Next(); isNextFrameValid; frame, isNextFrameValid = frames.Next() {
		if stringContainAnySubString(frame.Function, whiteListForFunctionsInBacktrace) {
			filenames = append(filenames, fmt.Sprintf("%s(%s:%d)", functionNameFromPath(frame.Function), frame.File, frame.Line))
		}
	}

	return filenames
}

type CustomLoggerFormatter struct {
	Hostname              string
	AppName               string
	InjectedToOtherLogger bool
	LogType               string
}

//Marshaler is an interface any type can implement to change its output in our production logs.
type Marshaler interface {
	MarshalLog() map[string]interface{}
}

func NewCustomLoggerFormatter(config *appconf.SourcesApiConfig, injectedToOtherLogger bool, logType string) *CustomLoggerFormatter {
	return &CustomLoggerFormatter{AppName: config.AppName, Hostname: config.Hostname, InjectedToOtherLogger: injectedToOtherLogger, LogType: logType}
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

	if entry.Level == logrus.ErrorLevel || entry.Level == logrus.WarnLevel {
		data["backtrace"] = backtrace()
	}

	if f.LogType != "" {
		data["log_type"] = f.LogType
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

func logLevelToEchoLogLevel(configLogLevel string) echoLog.Lvl {
	var logLevel echoLog.Lvl

	switch strings.ToUpper(configLogLevel) {
	case "DEBUG":
		logLevel = echoLog.DEBUG
	case "ERROR":
		logLevel = echoLog.ERROR
	case "WARN":
		logLevel = echoLog.WARN
	default:
		logLevel = echoLog.INFO
	}

	return logLevel
}

func FormatForMiddleware(config *appconf.SourcesApiConfig) string {
	// fields of default format from (converted to JSON): labstack/echo/v4@v4.4.0/middleware/logger.go
	defaultFormat := `{"time":"${time_rfc3339_nano}","id":"${id}","remote_ip":"${remote_ip}",` +
		`"host":"${host}","method":"${method}","uri":"${uri}","user_agent":"${user_agent}",` +
		`"status":"${status}","error":"${error}","latency":"${latency}","latency_human":"${latency_human}"` +
		`,"bytes_in":"${bytes_in}","bytes_out":"${bytes_out}"}` + "\n"

	fieldsDefaultFormat := make(map[string]interface{})

	err := json.Unmarshal([]byte(defaultFormat), &fieldsDefaultFormat)

	if err != nil {
		Log.Warn("Unmarshalling Error in FormatForMiddleware: " + err.Error())
		return defaultFormat
	}

	for k, v := range basicLogFields(config.LogLevelForMiddlewareLogs, config.AppName, config.Hostname) {
		fieldsDefaultFormat[k] = v
	}

	fieldsDefaultFormat["log_type"] = RequestType

	j, err := json.Marshal(fieldsDefaultFormat)

	if err != nil {
		Log.Warn("Marshalling Error in FormatForMiddleware: " + err.Error())
		return defaultFormat
	}

	return string(j)
}

func InitEchoLogger(e *echo.Echo, config *appconf.SourcesApiConfig) {
	logger := logrusEcho.Logger()
	logger.SetOutput(LogOutputFrom(config.LogHandler))
	logger.SetFormatter(NewCustomLoggerFormatter(config, true, EchoType))

	AddHooksTo(logger.Logger, config)
	e.Logger = logger
	e.Logger.SetLevel(logLevelToEchoLogLevel(config.LogLevel))
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

func InitLogger(config *appconf.SourcesApiConfig) {
	Log = &logrus.Logger{
		Out:          LogOutputFrom(config.LogHandler),
		Level:        LogrusLogLevelFrom(config.LogLevel),
		Formatter:    NewCustomLoggerFormatter(config, false, DefaultType),
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: true,
	}

	AddHooksTo(Log, config)
}
