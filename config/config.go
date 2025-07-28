package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"github.com/spf13/viper"
)

const (
	KafkaGroupId  = "sources-api-go"
	RdsCaLocation = "rdsca.cert"

	DatabaseStore       = "database"
	VaultStore          = "vault"
	SecretsManagerStore = "secrets-manager"
)

var parsedConfig *SourcesApiConfig

// SourcesApiConfig is the struct for storing runtime configuration
type SourcesApiConfig struct {
	AppName                  string
	Hostname                 string
	KafkaBrokerConfig        []clowder.BrokerConfig
	KafkaTopics              map[string]string
	KafkaGroupID             string
	MetricsPort              int
	LogLevel                 string
	LogGroup                 string
	AwsRegion                string
	AwsAccessKeyID           string
	AwsSecretAccessKey       string
	DatabaseHost             string
	DatabasePort             int
	DatabaseUser             string
	DatabasePassword         string
	DatabaseName             string
	DatabaseSSLMode          string
	DatabaseCert             string
	DisabledApplicationTypes []string
	FeatureFlagsEnvironment  string
	FeatureFlagsUrl          string
	FeatureFlagsAPIToken     string
	FeatureFlagsService      string
	CacheHost                string
	CachePort                int
	CachePassword            string
	SlowSQLThreshold         int
	AuthorizedPsks           []string
	BypassRbac               bool
	StatusListener           bool
	BackgroundWorker         bool
	MigrationsSetup          bool
	MigrationsReset          bool
	SecretStore              string
	TenantTranslatorUrl      string
	Env                      string
	HandleTenantRefresh      bool
	RbacHost                 string
	JWKSUrl                  string
	AuthorizedJWTSubjects    []string

	SecretsManagerAccessKey string
	SecretsManagerSecretKey string
	SecretsManagerPrefix    string
	LocalStackURL           string
}

// String() returns a string that shows the settings in which the pod is running in
func (s SourcesApiConfig) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s=%s ", "AppName", s.AppName)
	fmt.Fprintf(&b, "%s=%v ", "Hostname", s.Hostname)
	fmt.Fprintf(&b, "%s=%v ", "KafkaBrokerConfig", s.KafkaBrokerConfig)
	fmt.Fprintf(&b, "%s=%v ", "KafkaTopics", s.KafkaTopics)
	fmt.Fprintf(&b, "%s=%v ", "KafkaGroupID", s.KafkaGroupID)
	fmt.Fprintf(&b, "%s=%v ", "MetricsPort", s.MetricsPort)
	fmt.Fprintf(&b, "%s=%v ", "LogLevel", s.LogLevel)
	fmt.Fprintf(&b, "%s=%v ", "LogGroup", s.LogGroup)
	fmt.Fprintf(&b, "%s=%v ", "AwsRegion", s.AwsRegion)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseHost", s.DatabaseHost)
	fmt.Fprintf(&b, "%s=%v ", "DatabasePort", s.DatabasePort)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseName", s.DatabaseName)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseSSLMode", s.DatabaseSSLMode)
	fmt.Fprintf(&b, "%s=%v ", "DatabaseCert", s.DatabaseCert)
	fmt.Fprintf(&b, "%s=%v ", "DisabledApplicationTypes", s.DisabledApplicationTypes)
	fmt.Fprintf(&b, "%s=%v ", "FeatureFlagsEnvironment", s.FeatureFlagsEnvironment)
	fmt.Fprintf(&b, "%s=%v ", "FeatureFlagsUrl", s.FeatureFlagsUrl)
	fmt.Fprintf(&b, "%s=%v ", "FeatureFlagsService", s.FeatureFlagsService)
	fmt.Fprintf(&b, "%s=%v ", "CacheHost", s.CacheHost)
	fmt.Fprintf(&b, "%s=%v ", "CachePort", s.CachePort)
	fmt.Fprintf(&b, "%s=%v ", "SlowSQLThreshold", s.SlowSQLThreshold)
	fmt.Fprintf(&b, "%s=%v ", "BypassRbac", s.BypassRbac)
	fmt.Fprintf(&b, "%s=%v ", "SecretStore", s.SecretStore)
	fmt.Fprintf(&b, "%s=%v ", "TenantTranslatorUrl", s.TenantTranslatorUrl)
	fmt.Fprintf(&b, "%s=%v ", "Env", s.Env)
	fmt.Fprintf(&b, "%s=%v ", "HandleTenantRefresh", s.HandleTenantRefresh)
	fmt.Fprintf(&b, "%s=%v ", "SecretsManagerPrefix", s.SecretsManagerPrefix)
	fmt.Fprintf(&b, "%s=%v ", "LocalStackURL", s.LocalStackURL)
	fmt.Fprintf(&b, "%s=%v ", "RbacHost", s.RbacHost)
	fmt.Fprintf(&b, "%s=%v ", "JWKSUrl", s.JWKSUrl)

	return b.String()
}

// Reset clears the cached configuration - primarily for testing purposes
func Reset() {
	parsedConfig = nil
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

			options.SetDefault("FeatureFlagsAPIToken", clientAccessToken)
		}

		// Grab RBAC's host from Clowder's configuration...
		rbacEndpoint, err := findDependentApplication("rbac", cfg.Endpoints)
		if err != nil {
			log.Fatalf(`unable to read RBAC dependency's details from Clowder: %s`, err)
		}

		// ... and add the proper HTTP scheme depending on whether we have a non-zero TLS port or not.
		if rbacEndpoint.TlsPort != nil && *rbacEndpoint.TlsPort != 0 {
			options.SetDefault("RbacHost", fmt.Sprintf("https://%s:%d", rbacEndpoint.Hostname, rbacEndpoint.TlsPort))
		} else {
			options.SetDefault("RbacHost", fmt.Sprintf("http://%s:%d", rbacEndpoint.Hostname, rbacEndpoint.Port))
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

		options.SetDefault("RbacHost", os.Getenv("RBAC_HOST"))
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
	options.SetDefault("SlowSQLThreshold", 2) //seconds
	options.SetDefault("BypassRbac", os.Getenv("BYPASS_RBAC") == "true")

	switch os.Getenv("SECRET_STORE") {
	case SecretsManagerStore:
		secretManagerAccessKey := os.Getenv("SECRETS_MANAGER_ACCESS_KEY")
		secretManagerSecretKey := os.Getenv("SECRETS_MANAGER_SECRET_KEY")

		if secretManagerAccessKey == "" || secretManagerSecretKey == "" {
			log.Fatalf(`The AWS' secret manager store requires an access key and a secret key, but one of them is missing`)
		}

		options.SetDefault("LocalStackURL", os.Getenv("LOCALSTACK_URL"))
		options.SetDefault("SecretsManagerAccessKey", secretManagerAccessKey)
		options.SetDefault("SecretsManagerSecretKey", secretManagerSecretKey)

		prefix := os.Getenv("SECRETS_MANAGER_PREFIX")
		if prefix == "" {
			prefix = "sources-development"
		}

		options.SetDefault("SecretsManagerPrefix", prefix)
		options.SetDefault("SecretStore", os.Getenv("SECRET_STORE"))

	default:
		options.SetDefault("SecretStore", "database")
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
	options.SetDefault("AuthorizedPsks", strings.Split(os.Getenv("SOURCES_PSKS"), ","))

	// JWT authentication configuration
	options.SetDefault("JWKSUrl", os.Getenv("JWKS_URL"))
	options.SetDefault("AuthorizedJWTSubjects", strings.Split(os.Getenv("AUTHORIZED_JWT_SUBJECTS"), ","))

	// Grab the Kafka Sasl Settings.
	var brokerConfig []clowder.BrokerConfig

	bcRaw, ok := options.Get("KafkaBrokerConfig").([]clowder.BrokerConfig)
	if ok {
		brokerConfig = bcRaw
	}

	// Grab the disabled application types.
	disabledAppTypesEnv := os.Getenv("DISABLED_APPLICATION_TYPES")

	var disabledAppTypes []string
	if disabledAppTypesEnv != "" {
		disabledAppTypes = strings.Split(disabledAppTypesEnv, ",")
	} else {
		disabledAppTypes = []string{}
	}

	options.SetDefault("DisabledApplicationTypes", disabledAppTypes)

	// Parse all the configuration.
	options.AutomaticEnv()
	parsedConfig = &SourcesApiConfig{
		AppName:                  options.GetString("AppName"),
		Hostname:                 options.GetString("Hostname"),
		KafkaBrokerConfig:        brokerConfig,
		KafkaTopics:              options.GetStringMapString("KafkaTopics"),
		KafkaGroupID:             options.GetString("KafkaGroupID"),
		MetricsPort:              options.GetInt("MetricsPort"),
		LogLevel:                 options.GetString("LogLevel"),
		SlowSQLThreshold:         options.GetInt("SlowSQLThreshold"),
		LogGroup:                 options.GetString("LogGroup"),
		AwsRegion:                options.GetString("AwsRegion"),
		AwsAccessKeyID:           options.GetString("AwsAccessKeyID"),
		AwsSecretAccessKey:       options.GetString("AwsSecretAccessKey"),
		DatabaseHost:             options.GetString("DatabaseHost"),
		DatabasePort:             options.GetInt("DatabasePort"),
		DatabaseUser:             options.GetString("DatabaseUser"),
		DatabasePassword:         options.GetString("DatabasePassword"),
		DatabaseName:             options.GetString("DatabaseName"),
		DatabaseSSLMode:          options.GetString("DatabaseSSLMode"),
		DatabaseCert:             options.GetString("DatabaseCert"),
		DisabledApplicationTypes: options.GetStringSlice("DisabledApplicationTypes"),
		FeatureFlagsEnvironment:  options.GetString("FeatureFlagsEnvironment"),
		FeatureFlagsUrl:          options.GetString("FeatureFlagsUrl"),
		FeatureFlagsAPIToken:     options.GetString("FeatureFlagsAPIToken"),
		FeatureFlagsService:      options.GetString("FeatureFlagsService"),
		CacheHost:                options.GetString("CacheHost"),
		CachePort:                options.GetInt("CachePort"),
		CachePassword:            options.GetString("CachePassword"),
		AuthorizedPsks:           options.GetStringSlice("AuthorizedPsks"),
		BypassRbac:               options.GetBool("BypassRbac"),
		StatusListener:           options.GetBool("StatusListener"),
		BackgroundWorker:         options.GetBool("BackgroundWorker"),
		MigrationsSetup:          options.GetBool("MigrationsSetup"),
		MigrationsReset:          options.GetBool("MigrationsReset"),
		SecretStore:              options.GetString("SecretStore"),
		TenantTranslatorUrl:      options.GetString("TenantTranslatorUrl"),
		Env:                      options.GetString("Env"),
		HandleTenantRefresh:      options.GetBool("HandleTenantRefresh"),
		SecretsManagerAccessKey:  options.GetString("SecretsManagerAccessKey"),
		SecretsManagerSecretKey:  options.GetString("SecretsManagerSecretKey"),
		SecretsManagerPrefix:     options.GetString("SecretsManagerPrefix"),
		LocalStackURL:            options.GetString("LocalStackURL"),
		RbacHost:                 options.GetString("RbacHost"),
		JWKSUrl:                  options.GetString("JWKSUrl"),
		AuthorizedJWTSubjects:    options.GetStringSlice("AuthorizedJWTSubjects"),
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

// findDependentApplication finds the specified application in the Clowder configuration's endpoints section.
func findDependentApplication(name string, endpoints []clowder.DependencyEndpoint) (clowder.DependencyEndpoint, error) {
	idx := slices.IndexFunc[[]clowder.DependencyEndpoint](endpoints, func(endpoint clowder.DependencyEndpoint) bool {
		return strings.EqualFold(name, endpoint.App)
	})

	if idx == -1 {
		return clowder.DependencyEndpoint{}, fmt.Errorf(`unable to find application "%s" in the endpoints section of the cdappconfig.json file`, name)
	} else {
		return endpoints[idx], nil
	}
}
