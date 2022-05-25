package config

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	MarketplaceHost           string
	AwsRegion                 string
	AwsAccessKeyID            string
	AwsSecretAccessKey        string
	DatabaseHost              string
	DatabasePort              int
	DatabaseUser              string
	DatabasePassword          string
	DatabaseName              string
	FeatureFlagsEnvironment   string
	FeatureFlagsUrl           string
	FeatureFlagsAPIToken      string
	FeatureFlagsService       string
	FeatureFlagsBearerToken   string
	CacheHost                 string
	CachePort                 int
	CachePassword             string
	SlowSQLThreshold          int
	Psks                      []string
	BypassRbac                bool
	StatusListener            bool
	BackgroundWorker          bool
	MigrationsSetup           bool
	MigrationsReset           bool
	SecretStore               string
	TenantTranslatorUrl       string
}

// Get - returns the config parsed from runtime vars
func Get() *SourcesApiConfig {
	// check if we have already parsed the config - otherwise populate it.
	if parsedConfig != nil {
		return parsedConfig
	}

	options := viper.New()
	kafkaTopics := make(map[string]string)

	// Parse clowder, else the environment
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

		if cfg.FeatureFlags != nil {
			unleashUrl := ""
			if cfg.FeatureFlags.Hostname != "" && cfg.FeatureFlags.Port != 0 && cfg.FeatureFlags.Scheme != "" {
				unleashUrl = fmt.Sprintf("%s://%s:%d/api", cfg.FeatureFlags.Scheme, cfg.FeatureFlags.Hostname, cfg.FeatureFlags.Port)
			}

			options.SetDefault("FeatureFlagsUrl", unleashUrl)

			clientAccessToken := ""
			if cfg.FeatureFlags.ClientAccessToken != nil {
				clientAccessToken = *cfg.FeatureFlags.ClientAccessToken
			}
			options.SetDefault("FeatureFlagsBearerToken", clientAccessToken)
		}

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
		// Setting a default database name mimics the behaviour of the Rails app, which would set a default database
		// name for development in the case it wasn't overridden.
		databaseName := os.Getenv("DATABASE_NAME")
		if databaseName != "" {
			options.SetDefault("DatabaseName", databaseName)
		} else {
			options.SetDefault("DatabaseName", "sources_api_development")
		}
		options.SetDefault("CacheHost", os.Getenv("REDIS_CACHE_HOST"))
		options.SetDefault("CachePort", os.Getenv("REDIS_CACHE_PORT"))
		options.SetDefault("CachePassword", os.Getenv("REDIS_CACHE_PASSWORD"))

		options.SetDefault("FeatureFlagsUrl", os.Getenv("UNLEASH_URL"))
		options.SetDefault("FeatureFlagsAPIToken", os.Getenv("UNLEASH_TOKEN"))
	}

	options.SetDefault("FeatureFlagsService", os.Getenv("FEATURE_FLAGS_SERVICE"))

	if os.Getenv("SOURCES_ENV") == "prod" {
		options.SetDefault("FeatureFlagsEnvironment", "production")
	} else {
		options.SetDefault("FeatureFlagsEnvironment", "development")
	}

	options.SetDefault("KafkaGroupID", "sources-api-go")
	options.SetDefault("KafkaTopics", kafkaTopics)

	options.SetDefault("LogLevel", os.Getenv("LOG_LEVEL"))
	options.SetDefault("LogHandler", os.Getenv("LOG_HANDLER"))
	options.SetDefault("LogLevelForMiddlewareLogs", "DEBUG")
	options.SetDefault("LogLevelForSqlLogs", "DEBUG")
	options.SetDefault("MarketplaceHost", os.Getenv("MARKETPLACE_HOST"))
	options.SetDefault("SlowSQLThreshold", 2) //seconds
	options.SetDefault("BypassRbac", os.Getenv("BYPASS_RBAC") == "true")
	// The secret store defaults to the database in case an empty or an incorrect value are provided.
	secretStore := os.Getenv("SECRET_STORE")
	if secretStore != "database" && secretStore != "vault" {
		secretStore = "database"
	}
	options.SetDefault("SecretStore", secretStore)
	options.SetDefault("TenantTranslatorUrl", os.Getenv("TENANT_TRANSLATOR_URL"))

	// Parse any Flags (using our own flag set to not conflict with the global flag)
	fs := flag.NewFlagSet("runtime", flag.ContinueOnError)
	availabilityListener := fs.Bool("listener", false, "run availability status listener")
	backgroundWorker := fs.Bool("background-worker", false, "run background worker")
	setUpDatabase := fs.Bool("setup", false, "create the database and exit")
	resetDatabase := fs.Bool("reset", false, "drop the database, recreate it and exit")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing flags: %v\n", err)
	}

	options.SetDefault("StatusListener", *availabilityListener)
	options.SetDefault("BackgroundWorker", *backgroundWorker)
	options.SetDefault("MigrationsSetup", *setUpDatabase)
	options.SetDefault("MigrationsReset", *resetDatabase)

	// Hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	options.SetDefault("Hostname", hostname)
	options.SetDefault("AppName", "source-api-go")

	// psks for .... psk authentication
	options.SetDefault("psks", strings.Split(os.Getenv("SOURCES_PSKS"), ","))

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
		LogLevelForSqlLogs:        options.GetString("LogLevelForSqlLogs"),
		SlowSQLThreshold:          options.GetInt("SlowSQLThreshold"),
		LogHandler:                options.GetString("LogHandler"),
		LogGroup:                  options.GetString("LogGroup"),
		MarketplaceHost:           options.GetString("MarketplaceHost"),
		AwsRegion:                 options.GetString("AwsRegion"),
		AwsAccessKeyID:            options.GetString("AwsAccessKeyID"),
		AwsSecretAccessKey:        options.GetString("AwsSecretAccessKey"),
		DatabaseHost:              options.GetString("DatabaseHost"),
		DatabasePort:              options.GetInt("DatabasePort"),
		DatabaseUser:              options.GetString("DatabaseUser"),
		DatabasePassword:          options.GetString("DatabasePassword"),
		DatabaseName:              options.GetString("DatabaseName"),
		FeatureFlagsEnvironment:   options.GetString("FeatureFlagsEnvironment"),
		FeatureFlagsUrl:           options.GetString("FeatureFlagsUrl"),
		FeatureFlagsAPIToken:      options.GetString("FeatureFlagsAPIToken"),
		FeatureFlagsBearerToken:   options.GetString("FeatureFlagsBearerToken"),
		FeatureFlagsService:       options.GetString("FeatureFlagsService"),
		CacheHost:                 options.GetString("CacheHost"),
		CachePort:                 options.GetInt("CachePort"),
		CachePassword:             options.GetString("CachePassword"),
		Psks:                      options.GetStringSlice("psks"),
		BypassRbac:                options.GetBool("BypassRbac"),
		StatusListener:            options.GetBool("StatusListener"),
		BackgroundWorker:          options.GetBool("BackgroundWorker"),
		MigrationsSetup:           options.GetBool("MigrationsSetup"),
		MigrationsReset:           options.GetBool("MigrationsReset"),
		SecretStore:               options.GetString("SecretStore"),
		TenantTranslatorUrl:       options.GetString("TenantTranslatorUrl"),
	}

	return parsedConfig
}

func (sourceConfig *SourcesApiConfig) KafkaTopic(requestedTopic string) string {
	topic, found := sourceConfig.KafkaTopics[requestedTopic]
	if !found {
		topic = requestedTopic
	}

	return topic
}

// IsVaultOn returns true if the authentications are backed by Vault. False, if they are backed by the database.
func IsVaultOn() bool {
	return parsedConfig.SecretStore == "vault"
}
