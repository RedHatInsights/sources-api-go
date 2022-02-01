package dao

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"gorm.io/gorm"
	"sigs.k8s.io/yaml"
)

//go:embed seeds/*
var seedsFS embed.FS

func seedDatabase() error {
	l.Log.Infof("Seeding SourceType Table")
	err := seedSourceTypes()
	if err != nil {
		return err
	}

	l.Log.Infof("Seeding ApplicationType Table")
	err = seedApplicationTypes()
	if err != nil {
		return err
	}

	l.Log.Infof("Seeding MetaData Table")
	err = seedMetaData()
	if err != nil {
		return err
	}

	return nil
}

func seedSourceTypes() error {
	// parse the map of seeds
	seeds := make(sourceTypeSeedMap)
	// reading from the embedded fs for the seeds directory
	data, err := seedsFS.ReadFile("seeds/source_types.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	// load the existing source types from the database
	sourceTypes := loadSourceTypeSeeds()
	if sourceTypes == nil {
		return fmt.Errorf("failed to load source type seeds")
	}

	// the skip list is defined in the environment
	skipList := strings.Split(os.Getenv("SOURCE_TYPE_SKIP_LIST"), ",")

	// loop through all of the seeds
	// 1. creating the record if they don't exist
	// 2. updating the ones that do exist
	// 3. deleting the ones removed from the list
	for name, values := range seeds {
		if util.SliceContainsString(skipList, name) {
			l.Log.Infof("Skipping SourceType %v", name)
			continue
		}

		// find the source type in the hash - delete is a no-op if it is not there.
		st := sourceTypes[name]
		delete(sourceTypes, name)

		// if the source type was not found - we are creating it
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

		var result *gorm.DB
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
	for name := range sourceTypes {
		l.Log.Infof("Deleting SourceType %v", name)
		result := DB.Delete(sourceTypes[name])
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func seedApplicationTypes() error {
	seeds := make(applicationTypeSeedMap)
	// reading from the embedded fs for the seeds directory
	data, err := seedsFS.ReadFile("seeds/application_types.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	// load the existing application types from the database
	appTypes := loadApplicationTypeSeeds()
	if appTypes == nil {
		return fmt.Errorf("failed to load application type seeds")
	}

	// the skip list is defined in the environment
	skipList := strings.Split(os.Getenv("APPLICATION_TYPE_SKIP_LIST"), ",")

	for name, values := range seeds {
		if util.SliceContainsString(skipList, name) {
			l.Log.Infof("Skipping ApplicationType %v", name)
			continue
		}

		// find the application type in the hash - delete is a no-op if it is not there.
		at := appTypes[name]
		delete(appTypes, name)

		// if the app type was not found - we are creating it
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

		var result *gorm.DB
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
	for name := range appTypes {
		l.Log.Infof("Deleting ApplicationType %v", name)
		result := DB.Delete(appTypes[name])
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

func seedMetaData() error {
	// delete all seeds - the id's don't have to remain static.
	result := DB.Where("1=1").Delete(&m.MetaData{})
	if result.Error != nil {
		return result.Error
	}

	// load the application types once so we don't load them in a loop
	appTypes := loadApplicationTypeSeeds()

	err := seedSuperkeyMetadata(appTypes)
	if err != nil {
		return err
	}
	err = seedAppMetadata(appTypes)
	if err != nil {
		return err
	}

	return nil
}

func seedSuperkeyMetadata(appTypes map[string]*m.ApplicationType) error {
	seeds := make(superkeyMetadataSeedMap)
	// reading from the embedded fs for the seeds directory
	data, err := seedsFS.ReadFile("seeds/superkey_metadata.yml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &seeds)
	if err != nil {
		return err
	}

	for name, value := range seeds {
		// first find the application type we are going to set the metadata on
		appType, ok := appTypes[name]
		if !ok {
			return fmt.Errorf("failed find application type %v", name)
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
				ApplicationTypeID: appType.Id,
				Type:              "SuperKeyMetaData",
			}

			result := DB.Create(&metadata)
			if result.Error != nil {
				return result.Error
			}
		}
	}

	return nil
}

func seedAppMetadata(appTypes map[string]*m.ApplicationType) error {
	seeds := make(appMetadataSeedMap)
	// reading from the embedded fs for the seeds directory
	data, err := seedsFS.ReadFile("seeds/app_metadata.yml")
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
		appType, ok := appTypes[name]
		if !ok {
			return fmt.Errorf("failed find application type %v", name)
		}

		for key, value := range values {
			payload, err := json.Marshal(value)
			if err != nil {
				return err
			}

			m := m.MetaData{
				Name:              key,
				Payload:           payload,
				ApplicationTypeID: appType.Id,
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

// loads all the data from the source_types table into a map of the name and a
// pointer to the record
func loadSourceTypeSeeds() map[string]*m.SourceType {
	sourceTypes := make([]m.SourceType, 0)
	result := DB.Model(&m.SourceType{}).Scan(&sourceTypes)
	if result.Error != nil {
		return nil
	}

	// parse them into a map with keys name and value db type
	hash := make(map[string]*m.SourceType)
	for i, sourceType := range sourceTypes {
		hash[sourceType.Name] = &sourceTypes[i]
	}

	return hash
}

// loads all the data from the source_types table into a map of the name and a
// pointer to the record
func loadApplicationTypeSeeds() map[string]*m.ApplicationType {
	// load all the seeds from the db
	appTypes := make([]m.ApplicationType, 0)
	result := DB.Model(&m.ApplicationType{}).Scan(&appTypes)
	if result.Error != nil {
		return nil
	}

	// parse them into a map with keys name and value db type
	hash := make(map[string]*m.ApplicationType)
	for i, appType := range appTypes {
		hash[appType.Name] = &appTypes[i]
	}

	return hash
}
