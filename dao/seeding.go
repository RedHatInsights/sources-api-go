package dao

import (
	"encoding/json"
	"os"

	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"sigs.k8s.io/yaml"
)

const DEFAULT_SEEDS_DIR = "./dao/seeds/"

func seedDatabase() error {
	return seedDatabaseFromDirectory(DEFAULT_SEEDS_DIR)
}

func seedDatabaseFromDirectory(seedDir string) error {
	l.Log.Infof("Seeding SourceType Table")
	err := seedSourceTypes(seedDir)
	if err != nil {
		return err
	}

	l.Log.Infof("Seeding ApplicationType Table")
	err = seedApplicationTypes(seedDir)
	if err != nil {
		return err
	}

	l.Log.Infof("Seeding MetaData Table")
	err = seedMetaData(seedDir)
	if err != nil {
		return err
	}

	return nil
}

func seedSourceTypes(seedDir string) error {
	// parse the map of seeds
	seeds := make(sourceTypeSeedMap)
	data, err := os.ReadFile(seedDir + "source_types.yml")
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

func seedApplicationTypes(seedDir string) error {
	seeds := make(applicationTypeSeedMap)
	data, err := os.ReadFile(seedDir + "application_types.yml")
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

		dependentApplications, err := json.Marshal(values.DependentApplications)
		if err != nil {
			return err
		}

		supportedSourceTypes, err := json.Marshal(values.SupportedSourceTypes)
		if err != nil {
			return err
		}

		supportedAuthenticationTypes, err := json.Marshal(values.SupportedAuthenticationTypes)
		if err != nil {
			return err
		}

		at.DependentApplications = dependentApplications
		at.SupportedSourceTypes = supportedSourceTypes
		at.SupportedAuthenticationTypes = supportedAuthenticationTypes
		at.DisplayName = values.DisplayName
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

func seedMetaData(seedDir string) error {
	// delete all seeds - the id's don't have to remain static.
	result := DB.Where("1=1").Delete(&m.MetaData{})
	if result.Error != nil {
		return result.Error
	}

	err := seedSuperkeyMetadata(seedDir)
	if err != nil {
		return err
	}
	err = seedAppMetadata(seedDir)
	if err != nil {
		return err
	}

	return nil
}

func seedSuperkeyMetadata(seedDir string) error {
	seeds := make(superkeyMetadataSeedMap)
	data, err := os.ReadFile(seedDir + "superkey_metadata.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	for name, value := range seeds {
		// first find the application type we are going to set the metadata on
		var apptype m.ApplicationType
		result := DB.Where("name = ?", name).First(&apptype)
		if result.Error != nil {
			l.Log.Errorf("Failed to find application type %v", name)
			return result.Error
		}

		// create each "step" as a record in the db
		for _, values := range value.Steps {
			payload, err := json.Marshal(values.Payload)
			if err != nil {
				return err
			}
			substitutions, err := json.Marshal(values.Substitutions)
			if err != nil {
				return err
			}

			metadata := m.MetaData{
				Step:              values.Step,
				Name:              values.Name,
				Payload:           payload,
				Substitutions:     substitutions,
				ApplicationTypeID: apptype.Id,
				Type:              "SuperKeyMetaData",
			}

			result = DB.Create(&metadata)
			if result.Error != nil {
				return result.Error
			}
		}
	}

	return nil
}

func seedAppMetadata(seedDir string) error {
	seeds := make(appMetadataSeedMap)
	data, err := os.ReadFile(seedDir + "app_metadata.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	// defaulting to "ci" if no var is set.
	env, ok := os.LookupEnv("SOURCES_ENV")
	if !ok {
		l.Log.Infof("Defaulting SOURCES_ENV to ci")
		env = "ci"
	}

	for name, values := range seeds[env] {
		var apptype m.ApplicationType
		result := DB.Where("name = ?", name).First(&apptype)
		if result.Error != nil {
			l.Log.Errorf("Failed to find application type %v", name)
			return result.Error
		}

		for key, value := range values {
			payload, err := json.Marshal(value)
			if err != nil {
				return err
			}

			m := m.MetaData{
				Name:              key,
				Payload:           payload,
				ApplicationTypeID: apptype.Id,
				Type:              "AppMetaData",
			}

			result := DB.Create(&m)
			if result.Error != nil {
				return result.Error
			}
		}
	}

	return nil
}
