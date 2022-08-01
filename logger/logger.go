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
	lc "github.com/redhatinsights/platform-go-middlewares/logging/cloudwatch"
	"github.com/sirupsen/logrus"
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

func parseLogLevel(configLogLevel string) logrus.Level {
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

type LogFormatter struct {
	Hostname string
	AppName  string
	LogType  string
}

//Marshaler is an interface any type can implement to change its output in our production logs.
type Marshaler interface {
	MarshalLog() map[string]interface{}
}

func newLoggerFormatter(config *appconf.SourcesApiConfig) *LogFormatter {
	return &LogFormatter{AppName: config.AppName, Hostname: config.Hostname}
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

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	data := basicLogFields(entry.Level.String(), f.AppName, f.Hostname)
	data["message"] = entry.Message

	var caller string
	if entry.Caller == nil {
		caller = ""
	} else {
		caller = entry.Caller.Func.Name()
	}

	data["caller"] = caller

	if entry.Logger.Level == logrus.DebugLevel && entry.Caller != nil {
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

func InitLogger(config *appconf.SourcesApiConfig) {
	Log = &logrus.Logger{
		Out:          os.Stdout,
		Level:        parseLogLevel(config.LogLevel),
		Formatter:    newLoggerFormatter(config),
		Hooks:        make(logrus.LevelHooks),
		ReportCaller: true,
	}

	AddHooksTo(Log, config)
}
