package amazon

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

var (
	// base config, populated from initial setup connecting to the sources account
	sourcesConfig *aws.Config

	// error we can return if there were no configured credentials
	ErrNoCredentials = errors.New("no credentials set to connect to AWS Secrets Manager")

	conf = config.Get()
)

func NewSecretsManagerClient() (SecretsManagerClient, error) {
	// first time initialization
	if sourcesConfig == nil {
		if conf.SecretsManagerAccessKey == "" || conf.SecretsManagerSecretKey == "" || conf.SecretsManagerPrefix == "" {
			return nil, ErrNoCredentials
		}

		cfg, err := awsConfig.LoadDefaultConfig(context.Background(), awsConfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				conf.SecretsManagerAccessKey,
				conf.SecretsManagerSecretKey,
				"sources-api-go-sm",
			),
		))
		if err != nil {
			return nil, err
		}

		sourcesConfig = &cfg
	}

	return &SecretsManagerClientImpl{
		sm:     secretsmanager.NewFromConfig(*sourcesConfig),
		prefix: conf.SecretsManagerPrefix,
	}, nil
}

type SecretsManagerClient interface {
	GetSecret(arn string) (*string, error)
	CreateSecret(auth *model.Authentication, value string) (*string, error)
	UpdateSecret(arn string, value string) error
	DeleteSecret(arn string) error
}

type SecretsManagerClientImpl struct {
	sm *secretsmanager.Client
	// prefix, set by app-interface
	prefix string
}

func (s *SecretsManagerClientImpl) GetSecret(arn string) (*string, error) {
	secret, err := s.sm.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: &arn})
	if err != nil {
		return nil, err
	}

	return secret.SecretString, nil
}

func (s *SecretsManagerClientImpl) CreateSecret(auth *model.Authentication, value string) (*string, error) {
	secret, err := s.sm.CreateSecret(context.Background(), &secretsmanager.CreateSecretInput{
		Name:         util.StringRef(fmt.Sprintf("%s/%d/%s-%d", s.prefix, auth.TenantID, auth.ResourceType, auth.ResourceID)),
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

func (s *SecretsManagerClientImpl) UpdateSecret(arn string, value string) error {
	_, err := s.sm.UpdateSecret(context.Background(), &secretsmanager.UpdateSecretInput{
		SecretId:     &arn,
		SecretString: &value,
	})

	return err
}

func (s *SecretsManagerClientImpl) DeleteSecret(arn string) error {
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
