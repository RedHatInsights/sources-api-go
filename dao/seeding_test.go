package dao

import (
	"os"
	"strings"
	"testing"

	m "github.com/RedHatInsights/sources-api-go/model"
	"sigs.k8s.io/yaml"
)

func TestSeedingSourceTypes(t *testing.T) {
	if !flags.Integration {
		t.Skip("seeding tests only run during integration tests")
	}

	if DB == nil {
		t.Fatal("DB is nil - cannot continue test.")
	}

	var err error
	var seedsDir string
	if strings.HasSuffix(os.Getenv("PWD"), "dao") {
		seedsDir = "./seeds/"
	} else {
		seedsDir = DEFAULT_SEEDS_DIR
	}

	err = seedSourceTypes(seedsDir)
	if err != nil {
		t.Fatal(err)
	}

	bytes, _ := os.ReadFile(seedsDir + "source_types.yml")
	seeds := make(sourceTypeSeedMap)
	err = yaml.Unmarshal(bytes, &seeds)
	if err != nil {
		t.Fatal(err)
	}

	stypes := make([]m.SourceType, 0)
	result := DB.Model(&m.SourceType{}).Scan(&stypes)
	if result.Error != nil {
		t.Fatalf("failed to list sourcetypes: %v", result.Error)
	}

	if len(stypes) != len(seeds) {
		t.Errorf("Seeding did not match values, got %v expected %v", len(stypes), len(seeds))
	}
}

func TestSeedingApplicationTypes(t *testing.T) {
	if !flags.Integration {
		t.Skip("seeding tests only run during integration tests")
	}

	if DB == nil {
		t.Fatal("DB is nil - cannot continue test.")
	}

	var err error
	var seedsDir string
	if strings.HasSuffix(os.Getenv("PWD"), "dao") {
		seedsDir = "./seeds/"
	} else {
		seedsDir = DEFAULT_SEEDS_DIR
	}

	err = seedApplicationTypes(seedsDir)
	if err != nil {
		t.Fatal(err)
	}

	bytes, _ := os.ReadFile(seedsDir + "application_types.yml")
	seeds := make(applicationTypeSeedMap)
	err = yaml.Unmarshal(bytes, &seeds)
	if err != nil {
		t.Fatal(err)
	}

	atypes := make([]m.ApplicationType, 0)
	result := DB.Model(&m.ApplicationType{}).Scan(&atypes)
	if result.Error != nil {
		t.Fatalf("failed to list apptypes: %v", result.Error)
	}

	if len(atypes) != len(seeds) {
		t.Errorf("Seeding did not match values, got %v expected %v", len(atypes), len(seeds))
	}
}

func TestSeedingSuperkeyMetadata(t *testing.T) {
	if !flags.Integration {
		t.Skip("seeding tests only run during integration tests")
	}

	if DB == nil {
		t.Fatal("DB is nil - cannot continue test.")
	}

	var err error
	var seedsDir string
	if strings.HasSuffix(os.Getenv("PWD"), "dao") {
		seedsDir = "./seeds/"
	} else {
		seedsDir = DEFAULT_SEEDS_DIR
	}

	appTypes := loadApplicationTypeSeeds()

	err = seedSuperkeyMetadata(seedsDir, appTypes)
	if err != nil {
		t.Fatal(err)
	}

	bytes, _ := os.ReadFile(seedsDir + "superkey_metadata.yml")
	seeds := make(superkeyMetadataSeedMap)
	err = yaml.Unmarshal(bytes, &seeds)
	if err != nil {
		t.Fatal(err)
	}

	skeymdata := make([]m.MetaData, 0)
	result := DB.Model(&m.MetaData{}).
		Where("type = ?", SuperKeyMetaData).
		Scan(&skeymdata)
	if result.Error != nil {
		t.Fatalf("failed to list superkey: %v", result.Error)
	}

	count := 0
	for _, v := range seeds {
		count += len(v.Steps)
	}

	if len(skeymdata) != count {
		t.Errorf("Seeding did not match values, got %v expected %v", len(skeymdata), count)
	}
}

func TestSeedingApplicationMetadata(t *testing.T) {
	if !flags.Integration {
		t.Skip("seeding tests only run during integration tests")
	}

	if DB == nil {
		t.Fatal("DB is nil - cannot continue test.")
	}

	var err error
	var seedsDir string
	if strings.HasSuffix(os.Getenv("PWD"), "dao") {
		seedsDir = "./seeds/"
	} else {
		seedsDir = DEFAULT_SEEDS_DIR
	}

	appTypes := loadApplicationTypeSeeds()

	err = seedAppMetadata(seedsDir, appTypes)
	if err != nil {
		t.Fatal(err)
	}

	bytes, _ := os.ReadFile(seedsDir + "app_metadata.yml")
	seeds := make(appMetadataSeedMap)
	err = yaml.Unmarshal(bytes, &seeds)
	if err != nil {
		t.Fatal(err)
	}

	appmdata := make([]m.MetaData, 0)
	result := DB.Model(&m.MetaData{}).
		Where("type = ?", AppMetaData).
		Distinct("name").
		Scan(&appmdata)
	if result.Error != nil {
		t.Fatalf("failed to list appmetadata: %v", result.Error)
	}

	count := len(seeds["ci"])

	if len(appmdata) != count {
		t.Errorf("Seeding did not match values, got %v expected %v", len(appmdata), count)
	}
}
