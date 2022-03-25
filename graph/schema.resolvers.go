package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func (r *applicationResolver) ID(ctx context.Context, obj *model.Application) (string, error) {
	return strconv.Itoa(int(obj.ID)), nil
}

func (r *applicationResolver) ApplicationTypeID(ctx context.Context, obj *model.Application) (string, error) {
	return strconv.Itoa(int(obj.ApplicationTypeID)), nil
}

func (r *applicationResolver) AvailabilityStatus(ctx context.Context, obj *model.Application) (*string, error) {
	if obj.AvailabilityStatus.AvailabilityStatus == "" {
		return nil, nil
	}

	return &obj.AvailabilityStatus.AvailabilityStatus, nil
}

func (r *applicationResolver) Authentications(ctx context.Context, obj *model.Application) ([]*model.Authentication, error) {
	auths, _, err := dao.GetAuthenticationDao(&r.Resolver.TenantID).ListForApplication(obj.ID, 100, 0, []util.Filter{})
	out := make([]*model.Authentication, len(auths))
	for i := range auths {
		out[i] = &auths[i]
	}
	return out, err
}

func (r *applicationResolver) TenantID(ctx context.Context, obj *model.Application) (string, error) {
	return strconv.Itoa(int(r.Resolver.TenantID)), nil
}

func (r *authenticationResolver) AvailabilityStatus(ctx context.Context, obj *model.Authentication) (*string, error) {
	if obj.AvailabilityStatus.AvailabilityStatus == "" {
		return nil, nil
	}

	return &obj.AvailabilityStatus.AvailabilityStatus, nil
}

func (r *authenticationResolver) ResourceID(ctx context.Context, obj *model.Authentication) (string, error) {
	return strconv.Itoa(int(obj.ResourceID)), nil
}

func (r *authenticationResolver) TenantID(ctx context.Context, obj *model.Authentication) (string, error) {
	return strconv.Itoa(int(r.Resolver.TenantID)), nil
}

func (r *endpointResolver) ID(ctx context.Context, obj *model.Endpoint) (string, error) {
	return strconv.Itoa(int(obj.ID)), nil
}

func (r *endpointResolver) AvailabilityStatus(ctx context.Context, obj *model.Endpoint) (*string, error) {
	if obj.AvailabilityStatus.AvailabilityStatus == "" {
		return nil, nil
	}

	return &obj.AvailabilityStatus.AvailabilityStatus, nil
}

func (r *endpointResolver) Authentications(ctx context.Context, obj *model.Endpoint) ([]*model.Authentication, error) {
	auths, _, err := dao.GetAuthenticationDao(&r.Resolver.TenantID).ListForEndpoint(obj.ID, 100, 0, []util.Filter{})
	out := make([]*model.Authentication, len(auths))
	for i := range auths {
		out[i] = &auths[i]
	}
	return out, err
}

func (r *endpointResolver) TenantID(ctx context.Context, obj *model.Endpoint) (string, error) {
	return strconv.Itoa(int(r.Resolver.TenantID)), nil
}

func (r *queryResolver) Sources(ctx context.Context) ([]model.Source, error) {
	srces, _, err := dao.GetSourceDao(&r.TenantID).List(100, 0, []util.Filter{})
	return srces, err
}

func (r *sourceResolver) ID(ctx context.Context, obj *model.Source) (string, error) {
	return strconv.Itoa(int(obj.ID)), nil
}

func (r *sourceResolver) SourceTypeID(ctx context.Context, obj *model.Source) (string, error) {
	return strconv.Itoa(int(obj.SourceTypeID)), nil
}

func (r *sourceResolver) AvailabilityStatus(ctx context.Context, obj *model.Source) (*string, error) {
	if obj.AvailabilityStatus.AvailabilityStatus == "" {
		return nil, nil
	}

	return &obj.AvailabilityStatus.AvailabilityStatus, nil
}

func (r *sourceResolver) Authentications(ctx context.Context, obj *model.Source) ([]*model.Authentication, error) {
	auths, _, err := dao.GetAuthenticationDao(&r.Resolver.TenantID).ListForSource(obj.ID, 100, 0, []util.Filter{})
	out := make([]*model.Authentication, len(auths))
	for i := range auths {
		out[i] = &auths[i]
	}
	return out, err
}

func (r *sourceResolver) Endpoints(ctx context.Context, obj *model.Source) ([]*model.Endpoint, error) {
	endpts, _, err := dao.GetEndpointDao(&r.Resolver.TenantID).SubCollectionList(*obj, 100, 0, []util.Filter{})
	out := make([]*model.Endpoint, len(endpts))
	for i := range endpts {
		out[i] = &endpts[i]
	}
	return out, err
}

func (r *sourceResolver) Applications(ctx context.Context, obj *model.Source) ([]*model.Application, error) {
	apps, _, err := dao.GetApplicationDao(&r.Resolver.TenantID).SubCollectionList(*obj, 100, 0, []util.Filter{})
	out := make([]*model.Application, len(apps))
	for i := range apps {
		out[i] = &apps[i]
	}
	return out, err
}

func (r *sourceResolver) TenantID(ctx context.Context, obj *model.Source) (string, error) {
	return strconv.Itoa(int(r.Resolver.TenantID)), nil
}

// Application returns generated.ApplicationResolver implementation.
func (r *Resolver) Application() generated.ApplicationResolver { return &applicationResolver{r} }

// Authentication returns generated.AuthenticationResolver implementation.
func (r *Resolver) Authentication() generated.AuthenticationResolver {
	return &authenticationResolver{r}
}

// Endpoint returns generated.EndpointResolver implementation.
func (r *Resolver) Endpoint() generated.EndpointResolver { return &endpointResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Source returns generated.SourceResolver implementation.
func (r *Resolver) Source() generated.SourceResolver { return &sourceResolver{r} }

type applicationResolver struct{ *Resolver }
type authenticationResolver struct{ *Resolver }
type endpointResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type sourceResolver struct{ *Resolver }
