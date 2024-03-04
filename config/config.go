package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/spf13/viper"
)

const (
	KafkaGroupId  = "sources-api-go"
	RdsCaLocation = "/app/rdsca.cert"

	DatabaseStore       = "database"
	VaultStore          = "vault"
	SecretsManagerStore = "secrets-manager"
)

var parsedConfig *SourcesApiConfig

// SourcesApiConfig is the struct for storing runtime configuration
type SourcesApiConfig struct {
	AppName                 string
	Hostname                string
	KafkaBrokerConfig       []clowder.BrokerConfig
	KafkaTopics             map[string]string
	KafkaGroupID            string
	MetricsPort             int
	LogLevel                string
	LogGroup                string
	MarketplaceHost         string
	AwsRegion               string
	AwsAccessKeyID          string
	AwsSecretAccessKey      string
	DatabaseHost            string
	DatabasePort            int
	DatabaseUser            string
	DatabasePassword        string
	DatabaseName            string
	DatabaseSSLMode         string
	DatabaseCert            string
	FeatureFlagsEnvironment string
	FeatureFlagsUrl         string
	FeatureFlagsAPIToken    string
	FeatureFlagsService     string
	FeatureFlagsBearerToken string
	CacheHost               string
	CachePort               int
	CachePassword           string
	SlowSQLThreshold        int
	Psks                    []string
	BypassRbac              bool
	StatusListener          bool
	BackgroundWorker        bool
	MigrationsSetup         bool
	MigrationsReset         bool
	SecretStore             string
	TenantTranslatorUrl     string
	Env                     string
	HandleTenantRefresh     bool

	SecretsManagerAccessKey string
	SecretsManagerSecretKey string
	SecretsManagerPrefix    string
	LocalStackURL           string
}

// String() returns a string that shows the settings in which the pod is running in
func (s SourcesApiConfig) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s=%s ", "AppName", parsedConfig.AppName)
	fmt.Fprintf(&b, "%s=%v ", "Hostname", parsedConfig.Hostname)
	fmt.Fprintf(&b, "%s=%v ", "KafkaBrokerConfig", parsedConfig.KafkaBrokerConfig)
	fmt.Fprintf(&b, "%s=%v ", "KafkaTopics", parsedConfig.KafkaTopics)
	fmt.Fprintf(&b, "%s=%v ", "KafkaGroupID", parsedConfig.KafkaGroupID)
	fmt.Fprintf(&b, "%s=%v ", "MetricsPort", parsedConfig.MetricsPort)
	fmt.Fprintf(&b, "%s=%v ", "LogLevel", parsedConfig.LogLevel)
	fmt.Fprintf(&b, "%s=%v ", "LogGroup", parsedConfig.LogGroup)
	fmt.Fprintf(&b, "%s=%v ", "MarketplaceHost", parsedConfig.MarketplaceHost)
	fmt.Fprintf(&b, "%s=%v ", "AwsRegion", parsedConfig.AwsRegion)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseHost", parsedConfig.DatabaseHost)
	fmt.Fprintf(&b, "%s=%v ", "DatabasePort", parsedConfig.DatabasePort)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseName", parsedConfig.DatabaseName)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseSSLMode", parsedConfig.DatabaseSSLMode)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseCert", parsedConfig.DatabaseCert)
	fmt.Fprintf(&b, "%s=%v ", "FeatureFlagsEnvironment", parsedConfig.FeatureFlagsEnvironment)
	fmt.Fprintf(&b, "%s=%v ", "FeatureFlagsUrl", parsedConfig.FeatureFlagsUrl)
	fmt.Fprintf(&b, "%s=%v ", "FeatureFlagsService", parsedConfig.FeatureFlagsService)
	fmt.Fprintf(&b, "%s=%v ", "CacheHost", parsedConfig.CacheHost)
	fmt.Fprintf(&b, "%s=%v ", "CachePort", parsedConfig.CachePort)
	fmt.Fprintf(&b, "%s=%v ", "SlowSQLThreshold", parsedConfig.SlowSQLThreshold)
	fmt.Fprintf(&b, "%s=%v ", "BypassRbac", parsedConfig.BypassRbac)
	fmt.Fprintf(&b, "%s=%v ", "SecretStore", parsedConfig.SecretStore)
	fmt.Fprintf(&b, "%s=%v ", "TenantTranslatorUrl", parsedConfig.TenantTranslatorUrl)
	fmt.Fprintf(&b, "%s=%v ", "Env", parsedConfig.Env)
	fmt.Fprintf(&b, "%s=%v ", "HandleTenantRefresh", parsedConfig.HandleTenantRefresh)
	fmt.Fprintf(&b, "%s=%v ", "SecretsManagerPrefix", parsedConfig.SecretsManagerPrefix)
	fmt.Fprintf(&b, "%s=%v ", "LocalStackURL", parsedConfig.LocalStackURL)
	return b.String()
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

		// [Kafka]
		if len(cfg.Kafka.Brokers) < 1 {
			log.Fatalf(`No Kafka brokers were found in the Clowder configuration`)
		}

		// Grab the first broker
		options.SetDefault("KafkaBrokerConfig", cfg.Kafka.Brokers)
		// [/Kafka]

		options.SetDefault("LogGroup", cfg.Logging.Cloudwatch.LogGroup)
		options.SetDefault("MetricsPort", cfg.MetricsPort)

		options.SetDefault("DatabaseHost", cfg.Database.Hostname)
		options.SetDefault("DatabasePort", cfg.Database.Port)
		options.SetDefault("DatabaseUser", cfg.Database.Username)
		options.SetDefault("DatabasePassword", cfg.Database.Password)
		options.SetDefault("DatabaseName", cfg.Database.Name)
		options.SetDefault("DatabaseSSLMode", cfg.Database.SslMode)

		if cfg.Database.RdsCa != nil {
			err := os.WriteFile(RdsCaLocation, []byte(*cfg.Database.RdsCa), 0644)
			if err != nil {
				panic(err)
			}

			options.SetDefault("DatabaseCert", RdsCaLocation)
		}

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

		kafkaPort := os.Getenv("QUEUE_PORT")
		if kafkaPort != "" {
			port, err := strconv.Atoi(kafkaPort)
			if err != nil {
				log.Fatalf(`the provided "QUEUE_PORT", "%s",  is not a valid integer: %s`, kafkaPort, err)
			}

			brokerConfig := []clowder.BrokerConfig{{
				Hostname: os.Getenv("QUEUE_HOST"),
				Port:     &port,
			}}

			options.SetDefault("KafkaBrokerConfig", brokerConfig)
		}

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
		options.SetDefault("DatabaseSSLMode", "disable")

		options.SetDefault("CacheHost", os.Getenv("REDIS_CACHE_HOST"))
		options.SetDefault("CachePort", os.Getenv("REDIS_CACHE_PORT"))
		options.SetDefault("CachePassword", os.Getenv("REDIS_CACHE_PASSWORD"))

		options.SetDefault("FeatureFlagsUrl", os.Getenv("UNLEASH_URL"))
		options.SetDefault("FeatureFlagsAPIToken", os.Getenv("UNLEASH_TOKEN"))
	}

	options.SetDefault("Env", os.Getenv("SOURCES_ENV"))

	handleTenantRefresh, _ := strconv.ParseBool(os.Getenv("HANDLE_TENANT_REFRESH"))
	options.SetDefault("HandleTenantRefresh", handleTenantRefresh)

	options.SetDefault("FeatureFlagsService", os.Getenv("FEATURE_FLAGS_SERVICE"))

	if os.Getenv("SOURCES_ENV") == "prod" {
		options.SetDefault("FeatureFlagsEnvironment", "production")
	} else {
		options.SetDefault("FeatureFlagsEnvironment", "development")
	}

	options.SetDefault("KafkaGroupID", KafkaGroupId)
	options.SetDefault("KafkaTopics", kafkaTopics)

	options.SetDefault("LogLevel", os.Getenv("LOG_LEVEL"))
	options.SetDefault("MarketplaceHost", os.Getenv("MARKETPLACE_HOST"))
	options.SetDefault("SlowSQLThreshold", 2) //seconds
	options.SetDefault("BypassRbac", os.Getenv("BYPASS_RBAC") == "true")
	secretStore := os.Getenv("SECRET_STORE")
	// The secret store defaults to the database in case an empty string
	if secretStore == "" {
		secretStore = "database"
	}
	options.SetDefault("SecretStore", secretStore)
	switch secretStore {
	case SecretsManagerStore:
		options.SetDefault("SecretsManagerAccessKey", os.Getenv("SECRETS_MANAGER_ACCESS_KEY"))
		options.SetDefault("SecretsManagerSecretKey", os.Getenv("SECRETS_MANAGER_SECRET_KEY"))
		prefix := os.Getenv("SECRETS_MANAGER_PREFIX")
		if prefix == "" {
			prefix = "sources-development"
		}
		options.SetDefault("SecretsManagerPrefix", prefix)
		options.SetDefault("LocalStackURL", os.Getenv("LOCALSTACK_URL"))
	}

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

	// Grab the Kafka Sasl Settings.
	var brokerConfig []clowder.BrokerConfig
	bcRaw, ok := options.Get("KafkaBrokerConfig").([]clowder.BrokerConfig)
	if ok {
		brokerConfig = bcRaw
	}

	options.AutomaticEnv()
	parsedConfig = &SourcesApiConfig{
		AppName:                 options.GetString("AppName"),
		Hostname:                options.GetString("Hostname"),
		KafkaBrokerConfig:       brokerConfig,
		KafkaTopics:             options.GetStringMapString("KafkaTopics"),
		KafkaGroupID:            options.GetString("KafkaGroupID"),
		MetricsPort:             options.GetInt("MetricsPort"),
		LogLevel:                options.GetString("LogLevel"),
		SlowSQLThreshold:        options.GetInt("SlowSQLThreshold"),
		LogGroup:                options.GetString("LogGroup"),
		MarketplaceHost:         options.GetString("MarketplaceHost"),
		AwsRegion:               options.GetString("AwsRegion"),
		AwsAccessKeyID:          options.GetString("AwsAccessKeyID"),
		AwsSecretAccessKey:      options.GetString("AwsSecretAccessKey"),
		DatabaseHost:            options.GetString("DatabaseHost"),
		DatabasePort:            options.GetInt("DatabasePort"),
		DatabaseUser:            options.GetString("DatabaseUser"),
		DatabasePassword:        options.GetString("DatabasePassword"),
		DatabaseName:            options.GetString("DatabaseName"),
		DatabaseSSLMode:         options.GetString("DatabaseSSLMode"),
		DatabaseCert:            options.GetString("DatabaseCert"),
		FeatureFlagsEnvironment: options.GetString("FeatureFlagsEnvironment"),
		FeatureFlagsUrl:         options.GetString("FeatureFlagsUrl"),
		FeatureFlagsAPIToken:    options.GetString("FeatureFlagsAPIToken"),
		FeatureFlagsBearerToken: options.GetString("FeatureFlagsBearerToken"),
		FeatureFlagsService:     options.GetString("FeatureFlagsService"),
		CacheHost:               options.GetString("CacheHost"),
		CachePort:               options.GetInt("CachePort"),
		CachePassword:           options.GetString("CachePassword"),
		Psks:                    options.GetStringSlice("psks"),
		BypassRbac:              options.GetBool("BypassRbac"),
		StatusListener:          options.GetBool("StatusListener"),
		BackgroundWorker:        options.GetBool("BackgroundWorker"),
		MigrationsSetup:         options.GetBool("MigrationsSetup"),
		MigrationsReset:         options.GetBool("MigrationsReset"),
		SecretStore:             options.GetString("SecretStore"),
		TenantTranslatorUrl:     options.GetString("TenantTranslatorUrl"),
		Env:                     options.GetString("Env"),
		HandleTenantRefresh:     options.GetBool("HandleTenantRefresh"),
		SecretsManagerAccessKey: options.GetString("SecretsManagerAccessKey"),
		SecretsManagerSecretKey: options.GetString("SecretsManagerSecretKey"),
		SecretsManagerPrefix:    options.GetString("SecretsManagerPrefix"),
		LocalStackURL:           options.GetString("LocalStackURL"),
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
// DEPRECATED: should be using methods on the authentication object instead of checking the config directly.
func IsVaultOn() bool {
	return parsedConfig.SecretStore == "vault"
}
