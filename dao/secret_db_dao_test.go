package dao

import (
	"strings"
	"testing"

	"github.com/RedHatInsights/sources-api-go/internal/testutils"
	"github.com/RedHatInsights/sources-api-go/util"
)

func TestSecretListUserOwnership(t *testing.T) {
	testutils.SkipIfNotRunningIntegrationTests(t)
	testutils.SkipIfNotSecretStoreDatabase(t)

	schema := "user_ownership"
	SwitchSchema(schema)
	// Set the encryption key
	util.OverrideEncryptionKey(strings.Repeat("test", 8))

	userIDWithOwnRecords := "user_based_user"
	otherUserIDWithOwnRecords := "other_user"
	userIDWithoutOwnRecords := "another_user"

	userWithOwnRecords, err := CreateUserForUserID(userIDWithOwnRecords, tenantId)
	if err != nil {
		t.Error(err)
	}

	secret1, err := CreateSecretByName("Secret 1", &tenantId, &userWithOwnRecords.Id)
	if err != nil {
		t.Error(err)
	}

	otherUserWithOwnRecords, err := CreateUserForUserID(otherUserIDWithOwnRecords, tenantId)
	if err != nil {
		t.Error(err)
	}

	_, err = CreateSecretByName("Secret 2", &tenantId, &otherUserWithOwnRecords.Id)
	if err != nil {
		t.Error(err)
	}

	secretWithoutOwnership, err := CreateSecretByName("Secret 3", &tenantId, nil)
	if err != nil {
		t.Error(err)
	}

	/*
	  Test 1 - User can see own records and records without ownership
	*/
	requestParams := &RequestParams{TenantID: &userWithOwnRecords.TenantID, UserID: &userWithOwnRecords.Id}
	secretDaoWithUser := GetSecretDao(requestParams)

	secrets, _, err := secretDaoWithUser.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the secrets: %s`, err)
	}

	var secretsIDs []int64
	for _, secret := range secrets {
		secretsIDs = append(secretsIDs, secret.DbID)
	}

	expectedSecretsIDs := []int64{secret1.DbID, secretWithoutOwnership.DbID}

	if !util.ElementsInSlicesEqual(secretsIDs, expectedSecretsIDs) {
		t.Errorf("Expected secrets IDs %v are not same with obtained IDs: %v", expectedSecretsIDs, secretsIDs)
	}

	userWithOwnRecords, err = CreateUserForUserID(userIDWithoutOwnRecords, tenantId)
	if err != nil {
		t.Error(err)
	}

	/*
	  Test 2 - User without any ownership records can see only records without ownership
	*/
	requestParams = &RequestParams{TenantID: &userWithOwnRecords.TenantID, UserID: &userWithOwnRecords.Id}
	secretDaoWithUser = GetSecretDao(requestParams)

	secrets, _, err = secretDaoWithUser.List(100, 0, []util.Filter{})
	if err != nil {
		t.Errorf(`unexpected error when listing the secrets: %s`, err)
	}

	secretsIDs = []int64{}
	for _, secret := range secrets {
		secretsIDs = append(secretsIDs, secret.DbID)
	}

	expectedSecretsIDs = []int64{secretWithoutOwnership.DbID}

	if !util.ElementsInSlicesEqual(secretsIDs, expectedSecretsIDs) {
		t.Errorf("Expected secrets IDs %v are not same with obtained IDs: %v", expectedSecretsIDs, secretsIDs)
	}

	DropSchema(schema)
}
