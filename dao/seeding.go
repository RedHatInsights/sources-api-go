package dao

import (
	"encoding/json"
	"os"

	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"sigs.k8s.io/yaml"
)

const SEEDS_DIR = "./dao/seeds/"

type (
	sourceTypeSeed      map[string]m.SourceTypeSeed
	applicationTypeSeed map[string]m.ApplicationTypeSeed
	metadataSeed        map[string]m.MetaDataSeed
)

func seedDatabase() error {
	err := seedSourceTypes()
	if err != nil {
		return err
	}
	err = seedApplicationTypes()
	if err != nil {
		return err
	}
	err = seedMetaData()
	if err != nil {
		return err
	}

	return nil
}

func seedSourceTypes() error {
	// parse the map of seeds
	seeds := make(sourceTypeSeed)
	data, err := os.ReadFile(SEEDS_DIR + "source_types.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	// load all the seeds from the db
	sourceTypes := make([]m.SourceType, 0)
	result := DB.Model(&m.SourceType{}).Scan(&sourceTypes)
	if result.Error != nil {
		return result.Error
	}

	// loop through all of the seeds
	// 1. creating the record if they don't exist
	// 2. updating the ones that do exist
	// 3. deleting the ones removed from the list
	for name, values := range seeds {
		var st *m.SourceType

		// find an existing one in the list of source types
		for i, sourceType := range sourceTypes {
			if sourceType.Name == name {
				st = &sourceType

				// deleting the sourcetype out of the array since we're handling
				// it.
				sourceTypes[i] = sourceTypes[len(sourceTypes)-1]
				sourceTypes = sourceTypes[:len(sourceTypes)-1]
				break
			}
		}

		// if the source type was not found - create it
		if st == nil {
			l.Log.Debugf("New SourceType found %v", name)
			st = &m.SourceType{}
		}

		schema, err := json.Marshal(values.Schema)
		if err != nil {
			return err
		}

		// mark the fields as updated
		st.ProductName = values.ProductName
		st.IconUrl = values.IconURL
		st.Schema = schema
		st.Vendor = values.Vendor
		st.Name = name

		// if this is a new record we need to create rather than save.
		if st.Id == 0 {
			result = DB.Create(&st)
		} else {
			result = DB.Save(&st)
		}

		if result.Error != nil {
			return result.Error
		}
	}

	// if there were any sourcetypes left - they were removed from the seed file
	// and need deleted
	for _, sourceType := range sourceTypes {
		l.Log.Infof("Deleting SourceType %v", sourceType.Name)
		result := DB.Delete(&sourceType)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func seedApplicationTypes() error {
	// TODO: why is my struct not working. no idea.
	seeds := make(map[string]map[string]interface{})
	data, err := os.ReadFile(SEEDS_DIR + "application_types.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	// load all the seeds from the db
	appTypes := make([]m.ApplicationType, 0)
	result := DB.Model(&m.ApplicationType{}).Scan(&appTypes)
	if result.Error != nil {
		return result.Error
	}

	for name, values := range seeds {
		var at *m.ApplicationType

		// find an existing one in the list of application types
		for i, appType := range appTypes {
			if appType.Name == name {
				at = &appType

				// deleting the apptype out of the array since we're handling
				// it.
				appTypes[i] = appTypes[len(appTypes)-1]
				appTypes = appTypes[:len(appTypes)-1]
				break
			}
		}

		// if the app type was not found - create it
		if at == nil {
			l.Log.Debugf("New ApplicationType found %v", name)
			at = &m.ApplicationType{}
		}

		dependentApplications, err := json.Marshal(values["dependent_applications"])
		if err != nil {
			return err
		}

		supportedSourceTypes, err := json.Marshal(values["supported_source_types"])
		if err != nil {
			return err
		}

		supportedAuthenticationTypes, err := json.Marshal(values["supported_authentication_types"])
		if err != nil {
			return err
		}

		at.DependentApplications = dependentApplications
		at.SupportedSourceTypes = supportedSourceTypes
		at.SupportedAuthenticationTypes = supportedAuthenticationTypes
		if values["display_name"] != nil {
			at.DisplayName = values["display_name"].(string)
		}
		at.Name = name

		// if this is a new record we need to create rather than save.
		if at.Id == 0 {
			result = DB.Create(&at)
		} else {
			result = DB.Save(&at)
		}

		if result.Error != nil {
			return result.Error
		}
	}

	// if there were any applicationtypes left - they were removed from the seed file
	// and need deleted
	for _, appType := range appTypes {
		l.Log.Infof("Deleting ApplicationType %v", appType.Name)
		result := DB.Delete(&appType)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func seedMetaData() error {
	seeds := make(metadataSeed)
	data, err := os.ReadFile(SEEDS_DIR + "superkey_metadata.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	// TODO: delete entire metadata table and re-seed it every time (id's do not
	// need to remain static)
	return nil
}
