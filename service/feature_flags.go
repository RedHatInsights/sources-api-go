package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RedHatInsights/sources-api-go/config"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/Unleash/unleash-client-go/v3"
)

const appName = "sources-api"
const projectName = "default"
const refreshInterval = 60 // seconds
const metricsInterval = 60 // seconds

var conf = config.Get()

var ready = false

func featureFlagsServiceUnleash() bool {
	return conf.FeatureFlagsService == "unleash"
}

type FeatureFlagListener struct{}

func (l FeatureFlagListener) OnError(err error) {
	logging.Log.Errorf("unleash error: %v\n", err)
}

func (l FeatureFlagListener) OnWarning(warning error) {
	logging.Log.Warnf("unleash warning: %v\n", warning)
}

func (l FeatureFlagListener) OnReady() {
	ready = true
	logging.Log.Info("connection to unleash instance is ready")
}

func (l FeatureFlagListener) OnCount(_ string, _ bool) {
}

func (l FeatureFlagListener) OnSent(_ unleash.MetricsData) {
}

func (l FeatureFlagListener) OnRegistered(_ unleash.ClientData) {
}

func init() {
	if featureFlagsServiceUnleash() {
		logging.InitLogger(conf)

		if conf.FeatureFlagsAPIToken == "" {
			logging.Log.Warnf("FeatureFlagsAPIToken is empty")
		}

		authorizationHeader := ""
		if conf.FeatureFlagsBearerToken != "" {
			authorizationHeader = fmt.Sprintf("Bearer %s", conf.FeatureFlagsAPIToken)
		} else {
			authorizationHeader = conf.FeatureFlagsAPIToken
		}

		unleashConfig := []unleash.ConfigOption{unleash.WithAppName(appName),
			unleash.WithListener(&FeatureFlagListener{}),
			unleash.WithUrl(conf.FeatureFlagsUrl),
			unleash.WithEnvironment(conf.FeatureFlagsEnvironment),
			unleash.WithRefreshInterval(refreshInterval * time.Second),
			unleash.WithMetricsInterval(metricsInterval * time.Second),
			unleash.WithProjectName(projectName),
			unleash.WithCustomHeaders(http.Header{"Authorization": {authorizationHeader}})}

		err := unleash.Initialize(unleashConfig...)
		if err != nil {
			logging.Log.Errorf("unable to initialize unleash: %v", err.Error())
		}
	}
}

func FeatureEnabled(feature string) bool {
	if !featureFlagsServiceUnleash() {
		return false
	}

	if !ready {
		return false
	}

	return unleash.IsEnabled(feature)
}
