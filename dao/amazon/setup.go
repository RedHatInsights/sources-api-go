package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const defaultRegion = "us-east-1"

// set up our base secrets-manager configuration
func initConfig() error {
	var cfg aws.Config
	var err error

	// setting up a config that points at our localstack instance, basically overwriting the resolver to not use the AWS API but instead our local instance. Handy for Eph and local development!
	if conf.LocalStackURL != "" {
		cfg, err = localStackConfig()
	} else {
		cfg, err = amazonConfig()
	}
	// err was set from one of the 2 loadDefaultConfig calls above
	if err != nil {
		return err
	}

	sourcesConfig = &cfg
	return nil
}

func localStackConfig() (aws.Config, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{PartitionID: "aws", URL: conf.LocalStackURL, SigningRegion: "us-east-1"}, nil
	})

	return awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(defaultRegion),
		awsConfig.WithEndpointResolverWithOptions(customResolver),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			conf.SecretsManagerAccessKey,
			conf.SecretsManagerSecretKey,
			"sources-api-go-secret-manager-store",
		)),
	)
}

func amazonConfig() (aws.Config, error) {
	if conf.SecretsManagerAccessKey == "" || conf.SecretsManagerSecretKey == "" {
		return aws.Config{}, ErrNoCredentials
	}

	return awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(defaultRegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			conf.SecretsManagerAccessKey,
			conf.SecretsManagerSecretKey,
			"sources-api-go-secret-manager-store",
		)),
	)
}
