package amazon

import (
	"context"
	"fmt"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/google/uuid"
)

var (
	conf           = config.Get()
	secretsManager SecretsManagerClient
)

func NewSecretsManagerClient(localStackURL, accessKey, secretKey string) (SecretsManagerClient, error) {
	if secretsManager != nil {
		return secretsManager, nil
	}

	cfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithRegion("us-east-1"),
		awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				"sources-api-go-secret-manager-store",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf(`unable to load default configuration with "%s" as the default region: %w"`, "us-east-1", err)
	}

	var sm *secretsmanager.Client
	if localStackURL == "" {
		sm = secretsmanager.NewFromConfig(cfg)
	} else {
		sm = secretsmanager.NewFromConfig(cfg, func(o *secretsmanager.Options) {
			o.EndpointResolverV2 = newEndpointResolver(localStackURL)
		})
	}

	return &secretsManagerClientImpl{
		sm:     sm,
		prefix: conf.SecretsManagerPrefix,
	}, nil
}

type SecretsManagerClient interface {
	GetSecret(arn string) (*string, error)
	CreateSecret(auth *model.Authentication, value string) (*string, error)
	UpdateSecret(arn string, value string) error
	DeleteSecret(arn string) error
}

type secretsManagerClientImpl struct {
	sm *secretsmanager.Client
	// prefix, set by app-interface
	prefix string
}

func (s *secretsManagerClientImpl) GetSecret(arn string) (*string, error) {
	secret, err := s.sm.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: &arn})
	if err != nil {
		return nil, err
	}

	return secret.SecretString, nil
}

func (s *secretsManagerClientImpl) CreateSecret(auth *model.Authentication, value string) (*string, error) {
	// for unique names, just spin up a guid.
	guid := uuid.NewString()[0:8]

	secret, err := s.sm.CreateSecret(context.Background(), &secretsmanager.CreateSecretInput{
		Name:         util.StringRef(fmt.Sprintf("%s/%d/%s-%d-%s", s.prefix, auth.TenantID, auth.ResourceType, auth.ResourceID, guid)),
		SecretString: &value,
		Tags: []types.Tag{
			{Key: util.StringRef("tenant"), Value: util.StringRef(strconv.Itoa(int(auth.TenantID)))},
		},
	})
	if err != nil {
		return nil, err
	}

	return secret.ARN, nil
}

func (s *secretsManagerClientImpl) UpdateSecret(arn string, value string) error {
	_, err := s.sm.UpdateSecret(context.Background(), &secretsmanager.UpdateSecretInput{
		SecretId:     &arn,
		SecretString: &value,
	})

	return err
}

func (s *secretsManagerClientImpl) DeleteSecret(arn string) error {
	trueVal := true

	_, err := s.sm.DeleteSecret(context.Background(), &secretsmanager.DeleteSecretInput{
		SecretId: &arn,
		// TODO: configurable per tenant maybe if they want to hold onto their
		// secrets in their accounts for 7 days or something. For now just
		// nuking from orbit.
		ForceDeleteWithoutRecovery: &trueVal,
	})

	return err
}
