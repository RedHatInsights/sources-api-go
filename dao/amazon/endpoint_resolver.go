package amazon

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

type resolverV2 struct {
	defaultRegion string
	localStackURL string
}

// newEndpointResolver creates a structure that helps pointing the AWS SDK to
// a "localstack" instance when developing locally.
func newEndpointResolver(localStackURL string) secretsmanager.EndpointResolverV2 {
	return &resolverV2{
		defaultRegion: "us-east-1",
		localStackURL: localStackURL,
	}
}

func (r *resolverV2) ResolveEndpoint(ctx context.Context, params secretsmanager.EndpointParameters) (smithyendpoints.Endpoint, error) {
	params.Region = &r.defaultRegion
	params.Endpoint = &r.localStackURL

	return secretsmanager.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
}
