package service

import (
	"encoding/json"
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

	req := superkey.CreateRequest{
		IdentityHeader:  identity,
		TenantID:        application.Tenant.ExternalTenant,
		SourceID:        strconv.FormatInt(application.SourceID, 10),
		ApplicationID:   strconv.FormatInt(application.ID, 10),
		ApplicationType: dao.Static.GetApplicationTypeName(application.ApplicationTypeID),
		SuperKey:        "0", // look up superkey auth
		Provider:        provider,
		Extra:           extra,
		SuperKeySteps:   steps,
	}

	// produce message to kafka topic
	fmt.Println(req)
	return nil
}

func getApplicationSuperkeyMetaData(application *m.Application) ([]superkey.Step, error) {
	// fetch the metadata from the db (no tenancy required)
	mDB := dao.GetMetaDataDao()
	metadata, err := mDB.GetSuperKeySteps(application.ApplicationTypeID)
	if err != nil {
		return nil, err
	}

	steps := make([]superkey.Step, len(metadata))

	// parse the data brought back from the db into the superkey "step" struct
	for i, step := range metadata {
		substitutions := make(map[string]string)
		err := json.Unmarshal(step.Substitutions, &substitutions)
		if err != nil {
			return nil, err
		}

		steps[i] = superkey.Step{
			Step:          step.Step,
			Name:          step.Name,
			Payload:       string(step.Payload),
			Substitutions: substitutions,
		}
	}

	return steps, nil
}

func getExtraValues(application *m.Application, provider string) (map[string]string, error) {
	extra := make(map[string]string)

	switch provider {
	case "amazon":
		// fetch the account number for replacing in the iam payloads
		var mDB dao.MetaDataDao = &dao.MetaDataDaoImpl{}
		acct, err := mDB.GetSuperKeyAccountNumber(application.ApplicationTypeID)
		if err != nil {
			return nil, err
		}
		extra["account"] = acct

		// fetch the result_type for the application_type
		var atDB dao.ApplicationTypeDao = &dao.ApplicationTypeDaoImpl{}
		authType, err := atDB.GetSuperKeyResultType(application.ApplicationTypeID, provider)
		if err != nil {
			return nil, err
		}
		extra["result_type"] = authType

	default:
		return nil, fmt.Errorf("invalid provider for superkey %v", provider)
	}

	return extra, nil
}

// loads up the application as well as the associates we need for the superkey
// request
func loadApplication(application *m.Application) (*m.Application, error) {
	appDao := dao.GetApplicationDao(&application.TenantID)

	// re-pulling it from the db to make sure we have the full-version, as well
	// as preloading any relations necessary.
	app, err := appDao.GetByIdWithPreload(&application.ID, "Source", "Source.Tenant")
	if err != nil {
		return nil, err
	}

	return app, nil
}
