package config

import (
	"fmt"
	"os"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/spf13/viper"
)

var parsedConfig *SourcesApiConfig

// SourcesApiConfig is the struct for storing runtime configuration
type SourcesApiConfig struct {
	AppName                   string
	Hostname                  string
	KafkaBrokers              []string
	KafkaTopics               map[string]string
	KafkaGroupID              string
	MetricsPort               int
	LogLevel                  string
	LogLevelForMiddlewareLogs string
	LogGroup                  string
	LogHandler                string
	LogLevelForSqlLogs        string
	AwsRegion                 string
	AwsAccessKeyID            string
	AwsSecretAccessKey        string
	DatabaseHost              string
	DatabasePort              int
	DatabaseUser              string
	DatabasePassword          string
	DatabaseName              string
	CacheHost                 string
	CachePort                 int
	CachePassword             string
	SlowSQLThreshold          int
}

// Get - returns the config parsed from runtime vars
func Get() *SourcesApiConfig {
	// check if we have already parsed the config - otherwise populate it.
	if parsedConfig != nil {
		return parsedConfig
	}

	options := viper.New()
	kafkaTopics := make(map[string]string)

	if clowder.IsClowderEnabled() {
		cfg := clowder.LoadedConfig

		for requestedName, topicConfig := range clowder.KafkaTopics {
			kafkaTopics[requestedName] = topicConfig.Name
		}
		options.SetDefault("AwsRegion", cfg.Logging.Cloudwatch.Region)
		options.SetDefault("AwsAccessKeyId", cfg.Logging.Cloudwatch.AccessKeyId)
		options.SetDefault("AwsSecretAccessKey", cfg.Logging.Cloudwatch.SecretAccessKey)
		options.SetDefault("KafkaBrokers", []string{fmt.Sprintf("%s:%v", cfg.Kafka.Brokers[0].Hostname, *cfg.Kafka.Brokers[0].Port)})
		options.SetDefault("LogGroup", cfg.Logging.Cloudwatch.LogGroup)
		options.SetDefault("MetricsPort", cfg.MetricsPort)

		options.SetDefault("DatabaseHost", cfg.Database.Hostname)
		options.SetDefault("DatabasePort", cfg.Database.Port)
		options.SetDefault("DatabaseUser", cfg.Database.Username)
		options.SetDefault("DatabasePassword", cfg.Database.Password)
		options.SetDefault("DatabaseName", cfg.Database.Name)

		options.SetDefault("CacheHost", cfg.InMemoryDb.Hostname)
		options.SetDefault("CachePort", cfg.InMemoryDb.Port)
		options.SetDefault("CachePassword", cfg.InMemoryDb.Password)
	} else {
		options.SetDefault("AwsRegion", "us-east-1")
		options.SetDefault("AwsAccessKeyId", os.Getenv("CW_AWS_ACCESS_KEY_ID"))
		options.SetDefault("AwsSecretAccessKey", os.Getenv("CW_AWS_SECRET_ACCESS_KEY"))
		options.SetDefault("KafkaBrokers", []string{fmt.Sprintf("%v:%v", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT"))})
		options.SetDefault("LogGroup", os.Getenv("CLOUD_WATCH_LOG_GROUP"))
		options.SetDefault("MetricsPort", 9394)

		options.SetDefault("DatabaseHost", os.Getenv("DATABASE_HOST"))
		options.SetDefault("DatabasePort", os.Getenv("DATABASE_PORT"))
		options.SetDefault("DatabaseUser", os.Getenv("DATABASE_USER"))
		options.SetDefault("DatabasePassword", os.Getenv("DATABASE_PASSWORD"))
		options.SetDefault("DatabaseName", os.Getenv("DATABASE_NAME"))

		options.SetDefault("CacheHost", os.Getenv("REDIS_CACHE_HOST"))
		options.SetDefault("CachePort", os.Getenv("REDIS_CACHE_PORT"))
		options.SetDefault("CachePassword", os.Getenv("REDIS_CACHE_PASSWORD"))
	}

	options.SetDefault("KafkaGroupID", "sources-api-go")
	options.SetDefault("KafkaTopics", kafkaTopics)

	options.SetDefault("LogLevel", os.Getenv("LOG_LEVEL"))
	options.SetDefault("LogHandler", os.Getenv("LOG_HANDLER"))
	options.SetDefault("LogLevelForMiddlewareLogs", "DEBUG")
	options.SetDefault("LogLevelForSqlLogs", "DEBUG")
	options.SetDefault("SlowSQLThreshold", 2) //seconds

	var (
		err      error
		hostname string
	)

	if hostname, err = os.Hostname(); err != nil {
		hostname = "unknown"
	}

	options.SetDefault("Hostname", hostname)
	options.SetDefault("AppName", "source-api-go")

	options.AutomaticEnv()
	parsedConfig = &SourcesApiConfig{
		AppName:                   options.GetString("AppName"),
		Hostname:                  options.GetString("Hostname"),
		KafkaBrokers:              options.GetStringSlice("KafkaBrokers"),
		KafkaTopics:               options.GetStringMapString("KafkaTopics"),
		KafkaGroupID:              options.GetString("KafkaGroupID"),
		MetricsPort:               options.GetInt("MetricsPort"),
		LogLevel:                  options.GetString("LogLevel"),
		LogLevelForMiddlewareLogs: options.GetString("LogLevelForMiddlewareLogs"),
		LogHandler:                options.GetString("LogHandler"),
		LogGroup:                  options.GetString("LogGroup"),
		AwsRegion:                 options.GetString("AwsRegion"),
		AwsAccessKeyID:            options.GetString("AwsAccessKeyID"),
		AwsSecretAccessKey:        options.GetString("AwsSecretAccessKey"),
		DatabaseHost:              options.GetString("DatabaseHost"),
		DatabasePort:              options.GetInt("DatabasePort"),
		DatabaseUser:              options.GetString("DatabaseUser"),
		DatabasePassword:          options.GetString("DatabasePassword"),
		DatabaseName:              options.GetString("DatabaseName"),
		CacheHost:                 options.GetString("CacheHost"),
		CachePort:                 options.GetInt("CachePort"),
		CachePassword:             options.GetString("CachePassword"),
	}

	return parsedConfig
}
