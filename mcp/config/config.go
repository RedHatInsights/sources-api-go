package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          int
	MetricsPort   int
	SourcesAPIURL string
	LogLevel      string
}

var config *Config

// Get returns the singleton configuration instance
func Get() *Config {
	if config != nil {
		return config
	}

	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	metricsPort := 9000
	if p := os.Getenv("METRICS_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			metricsPort = parsed
		}
	}

	sourcesAPIURL := os.Getenv("SOURCES_API_URL")
	if sourcesAPIURL == "" {
		sourcesAPIURL = "http://sources-api-svc:8000"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	config = &Config{
		Port:          port,
		MetricsPort:   metricsPort,
		SourcesAPIURL: sourcesAPIURL,
		LogLevel:      logLevel,
	}

	return config
}
