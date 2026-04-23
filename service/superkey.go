package service

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

const superkeyRequestedTopic = "platform.sources.superkey-requests"

var superkeyTopic = config.Get().KafkaTopic(superkeyRequestedTopic)

var (
	superkeyWriter     *kafka.Writer
	superkeyWriterOnce sync.Once
	superkeyWriterErr  error
)

// getSuperkeyWriter lazily initializes and returns the shared Kafka writer for
// superkey requests. Using a long-lived writer ensures the internal round-robin
// partitioner distributes messages across all partitions instead of always
// hitting the same one (which happens when a new writer is created per message).
func getSuperkeyWriter() (*kafka.Writer, error) {
	superkeyWriterOnce.Do(func() {
		superkeyWriter, superkeyWriterErr = kafka.GetWriter(&kafka.Options{
			BrokerConfig: conf.KafkaBrokerConfig,
			Topic:        superkeyTopic,
			Logger:       l.Log,
		})
	})

	return superkeyWriter, superkeyWriterErr
}

// CloseSuperkeyProducer closes the shared Kafka writer for superkey requests.
// Call during graceful shutdown.
func CloseSuperkeyProducer() {
	kafka.CloseWriter(superkeyWriter, "superkey producer shutdown")
}

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

	superKeyId := superKey.GetID()

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

	superKeyId := superKey.GetID()

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
	writer, err := getSuperkeyWriter()
	if err != nil {
		return fmt.Errorf(`unable to get the Kafka writer to produce a superkey request: %w`, err)
	}

	return kafka.Produce(writer, m)
}
