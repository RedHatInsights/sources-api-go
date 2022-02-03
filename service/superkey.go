package service

import (
	"fmt"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/redhatinsights/sources-superkey-worker/superkey"
)

func SendSuperKeyCreateRequest(identity string, application *m.Application) error {
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

	req := superkey.CreateRequest{
		// IdentityHeader:  identity, <- header
		TenantID:        application.Tenant.ExternalTenant,
		SourceID:        strconv.FormatInt(application.SourceID, 10),
		ApplicationID:   strconv.FormatInt(application.ID, 10),
		ApplicationType: dao.Static.GetApplicationTypeName(application.ApplicationTypeID),
		SuperKey:        superKey.ID,
		Provider:        provider,
		Extra:           extra,
		SuperKeySteps:   steps,
	}

	// produce message to kafka topic
	fmt.Println(req)
	return nil
}

func SendSuperKeyDeleteRequest(application *m.Application) error {
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
		return err
	}

	// parse out the existing data, we need to know the resource names to delete
	skData, err := parseSuperKeyData(application.SuperkeyData)
	if err != nil {
		return err
	}

	req := superkey.DestroyRequest{
		TenantID:       strconv.FormatInt(application.TenantID, 10),
		SuperKey:       superKey.ID,
		GUID:           skData.GUID,
		Provider:       skData.Provider,
		StepsCompleted: skData.StepsCompleted,
		SuperKeySteps:  steps,
	}

	fmt.Printf("req: %v\n", req)
	return nil
}
