package service

import (
	"strconv"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

const SUPERKEY_REQUEST_QUEUE = "platform.sources.superkey-requests"

func SendSuperKeyCreateRequest(application *m.Application, headers []kafka.Header) error {
	// load up the app + associations from the db+vault
	application, err := loadApplication(application)
	if err != nil {
		return err
	}

	// fetch the metadata and transform it
	steps, err := getApplicationSuperkeyMetaData(application)
	if err != nil {
		return err
	}

	// fetch the provider name from the static cache
	provider := dao.Static.GetSourceTypeName(application.Source.SourceTypeID)

	// fetch the extra values for this superkey request based on the provider type
	extra, err := getExtraValues(application, provider)
	if err != nil {
		return err
	}

	// fetch the superkey authentication
	superKey, err := getSuperKeyAuthentication(application)
	if err != nil {
		return err
	}

	var superKeyId string
	if config.IsVaultOn() {
		superKeyId = superKey.ID
	} else {
		superKeyId = strconv.FormatInt(superKey.DbID, 10)
	}

	req := superkey.CreateRequest{
		TenantID:        application.Tenant.ExternalTenant,
		SourceID:        strconv.FormatInt(application.SourceID, 10),
		ApplicationID:   strconv.FormatInt(application.ID, 10),
		ApplicationType: dao.Static.GetApplicationTypeName(application.ApplicationTypeID),
		SuperKey:        superKeyId,
		Provider:        provider,
		Extra:           extra,
		SuperKeySteps:   steps,
	}

	m := kafka.Message{}
	err = m.AddValueAsJSON(&req)
	if err != nil {
		return err
	}

	m.AddHeaders(append(headers, kafka.Header{Key: "event_type", Value: []byte("create_application")}))

	return produceSuperkeyRequest(&m)
}

func SendSuperKeyDeleteRequest(application *m.Application, headers []kafka.Header) error {
	// load up the app + associations from the db+vault
	application, err := loadApplication(application)
	if err != nil {
		return err
	}

	// fetch the metadata and transform it
	steps, err := getApplicationSuperkeyMetaData(application)
	if err != nil {
		return err
	}

	// grab the authentication required for hitting the superkey provider
	superKey, err := getSuperKeyAuthentication(application)
	if err != nil {
		l.Log.Warnf("SuperKey Authentication was nil - cleaning up incomplete superkey")
		return nil
	}

	// parse out the existing data, we need to know the resource names to delete
	skData, err := parseSuperKeyData(application.SuperkeyData)
	if err != nil {
		return err
	}
	if skData == nil {
		l.Log.Warnf("SuperKey Data was nil - cleaning up incomplete superkey")
		return nil
	}

	var superKeyId string
	if config.IsVaultOn() {
		superKeyId = superKey.ID
	} else {
		superKeyId = strconv.FormatInt(superKey.DbID, 10)
	}

	req := superkey.DestroyRequest{
		TenantID:       application.Tenant.ExternalTenant,
		SuperKey:       superKeyId,
		GUID:           skData.GUID,
		Provider:       skData.Provider,
		StepsCompleted: skData.StepsCompleted,
		SuperKeySteps:  steps,
	}

	m := kafka.Message{}
	err = m.AddValueAsJSON(&req)
	if err != nil {
		return err
	}

	m.AddHeaders(append(headers, kafka.Header{Key: "event_type", Value: []byte("destroy_application")}))

	return produceSuperkeyRequest(&m)
}

func produceSuperkeyRequest(m *kafka.Message) error {
	mgr := kafka.Manager{
		Config: kafka.Config{
			KafkaBrokers:   config.Get().KafkaBrokers,
			ProducerConfig: kafka.ProducerConfig{Topic: config.Get().KafkaTopic(SUPERKEY_REQUEST_QUEUE)}}}

	err := mgr.Produce(m)
	if err != nil {
		mgr.Producer().Close()
		return err
	}

	return mgr.Producer().Close()
}
