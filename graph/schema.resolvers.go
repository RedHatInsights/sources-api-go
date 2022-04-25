package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/RedHatInsights/sources-api-go/config"
	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/graph/generated"
	generated_model "github.com/RedHatInsights/sources-api-go/graph/model"
	"github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/util"
)

func (r *applicationResolver) ID(ctx context.Context, obj *model.Application) (string, error) {
	return strconv.Itoa(int(obj.ID)), nil
}

func (r *applicationResolver) ApplicationTypeID(ctx context.Context, obj *model.Application) (string, error) {
	return strconv.Itoa(int(obj.ApplicationTypeID)), nil
}

func (r *applicationResolver) Extra(ctx context.Context, obj *model.Application) (interface{}, error) {
	if obj.Extra == nil {
		return nil, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal(obj.Extra, &m)
	return m, err
}

func (r *applicationResolver) Authentications(ctx context.Context, obj *model.Application) ([]*model.Authentication, error) {
	err := getRequestDataFromCtx(ctx).EnsureAuthenticationsAreLoaded()
	if err != nil {
		return nil, err
	}

	auths := authenticationsFromCtx(ctx, "Application", obj.ID)
	out := make([]*model.Authentication, len(auths))
	for i := range auths {
		out[i] = &auths[i]
	}
	return out, err
}

func (r *applicationResolver) TenantID(ctx context.Context, obj *model.Application) (string, error) {
	return strconv.Itoa(int(*tenantIdFromCtx(ctx))), nil
}

func (r *applicationTypeResolver) ID(ctx context.Context, obj *model.ApplicationType) (string, error) {
	return strconv.FormatInt(obj.Id, 10), nil
}

func (r *applicationTypeResolver) DependentApplications(ctx context.Context, obj *model.ApplicationType) (interface{}, error) {
	return obj.DependentApplications, nil
}

func (r *applicationTypeResolver) SupportedSourceTypes(ctx context.Context, obj *model.ApplicationType) (interface{}, error) {
	return obj.SupportedSourceTypes, nil
}

func (r *applicationTypeResolver) SupportedAuthenticationTypes(ctx context.Context, obj *model.ApplicationType) (interface{}, error) {
	return obj.SupportedAuthenticationTypes, nil
}

func (r *applicationTypeResolver) Sources(ctx context.Context, obj *model.ApplicationType) ([]*model.Source, error) {
	srces, _, err := dao.GetSourceDao(tenantIdFromCtx(ctx)).SubCollectionList(model.ApplicationType{Id: obj.Id}, 500, 0, []util.Filter{})
	out := make([]*model.Source, len(srces))
	for i := range srces {
		out[i] = &srces[i]
	}

	return out, err
}

func (r *authenticationResolver) ID(ctx context.Context, obj *model.Authentication) (string, error) {
	if config.IsVaultOn() {
		return obj.ID, nil
	} else {
		return strconv.FormatInt(obj.DbID, 10), nil
	}
}

func (r *authenticationResolver) ResourceID(ctx context.Context, obj *model.Authentication) (string, error) {
	return strconv.Itoa(int(obj.ResourceID)), nil
}

func (r *authenticationResolver) TenantID(ctx context.Context, obj *model.Authentication) (string, error) {
	return strconv.Itoa(int(*tenantIdFromCtx(ctx))), nil
}

func (r *endpointResolver) ID(ctx context.Context, obj *model.Endpoint) (string, error) {
	return strconv.Itoa(int(obj.ID)), nil
}

func (r *endpointResolver) Authentications(ctx context.Context, obj *model.Endpoint) ([]*model.Authentication, error) {
	err := getRequestDataFromCtx(ctx).EnsureAuthenticationsAreLoaded()
	if err != nil {
		return nil, err
	}

	auths := authenticationsFromCtx(ctx, "Endpoint", obj.ID)
	out := make([]*model.Authentication, len(auths))
	for i := range auths {
		out[i] = &auths[i]
	}
	return out, err
}

func (r *endpointResolver) TenantID(ctx context.Context, obj *model.Endpoint) (string, error) {
	return strconv.Itoa(int(*tenantIdFromCtx(ctx))), nil
}

func (r *queryResolver) Sources(ctx context.Context, limit *int, offset *int, sortBy []*generated_model.SortBy, filter []*generated_model.Filter) ([]*model.Source, error) {
	// default limit and offset
	if limit == nil {
		limit = new(int)
		*limit = 100
	}
	if offset == nil {
		offset = new(int)
		*offset = 0
	}

	// parse any filters passed along the request
	f := parseArgs(sortBy, filter)

	// list the sources with filters en tote!
	srces, count, err := dao.GetSourceDao(tenantIdFromCtx(ctx)).List(*limit, *offset, f)
	sendCount(ctx, count)

	// storing the IDs of relevant sources on the request context for later subresources
	sourceIDs := make([]string, len(srces))

	// output data needs to be pointers for some reason.
	out := make([]*model.Source, len(srces))
	for i := range srces {
		out[i] = &srces[i]
		sourceIDs[i] = strconv.FormatInt(srces[i].ID, 10)
	}

	// this will unlock the source mutex and let the subresources go
	getRequestDataFromCtx(ctx).SetSourceIDs(sourceIDs)
	return out, err
}

func (r *queryResolver) ApplicationTypes(ctx context.Context, limit *int, offset *int, sortBy []*generated_model.SortBy, filter []*generated_model.Filter) ([]*model.ApplicationType, error) {
	// default limit and offset
	if limit == nil {
		limit = new(int)
		*limit = 100
	}
	if offset == nil {
		offset = new(int)
		*offset = 0
	}

	// parse any filters passed along the request
	f := parseArgs(sortBy, filter)
	appTypes, count, err := dao.GetApplicationTypeDao(tenantIdFromCtx(ctx)).List(*limit, *offset, f)
	sendCount(ctx, count)

	out := make([]*model.ApplicationType, len(appTypes))
	for i := range appTypes {
		out[i] = &appTypes[i]
	}
	return out, err
}

func (r *queryResolver) Meta(ctx context.Context) (*generated_model.Meta, error) {
	return &generated_model.Meta{Count: getCount(ctx)}, nil
}

func (r *sourceResolver) ID(ctx context.Context, obj *model.Source) (string, error) {
	return strconv.Itoa(int(obj.ID)), nil
}

func (r *sourceResolver) SourceTypeID(ctx context.Context, obj *model.Source) (string, error) {
	return strconv.Itoa(int(obj.SourceTypeID)), nil
}

func (r *sourceResolver) Authentications(ctx context.Context, obj *model.Source) ([]*model.Authentication, error) {
	err := getRequestDataFromCtx(ctx).EnsureAuthenticationsAreLoaded()
	if err != nil {
		return nil, err
	}

	auths := authenticationsFromCtx(ctx, "Source", obj.ID)
	out := make([]*model.Authentication, len(auths))
	for i := range auths {
		out[i] = &auths[i]
	}
	return out, err
}

func (r *sourceResolver) Endpoints(ctx context.Context, obj *model.Source) ([]*model.Endpoint, error) {
	err := getRequestDataFromCtx(ctx).EnsureEndpointsAreLoaded()
	if err != nil {
		return nil, err
	}

	endpts := sourceEndpointsFromCtx(ctx, obj.ID)
	out := make([]*model.Endpoint, len(endpts))
	for i := range endpts {
		out[i] = &endpts[i]
	}

	return out, nil
}

func (r *sourceResolver) Applications(ctx context.Context, obj *model.Source) ([]*model.Application, error) {
	err := getRequestDataFromCtx(ctx).EnsureApplicationsAreLoaded()
	if err != nil {
		return nil, err
	}

	apps := sourceApplicationsFromCtx(ctx, obj.ID)
	out := make([]*model.Application, len(apps))
	for i := range apps {
		out[i] = &apps[i]
	}

	return out, nil
}

func (r *sourceResolver) TenantID(ctx context.Context, obj *model.Source) (string, error) {
	return strconv.Itoa(int(*tenantIdFromCtx(ctx))), nil
}

// Application returns generated.ApplicationResolver implementation.
func (r *Resolver) Application() generated.ApplicationResolver { return &applicationResolver{r} }

// ApplicationType returns generated.ApplicationTypeResolver implementation.
func (r *Resolver) ApplicationType() generated.ApplicationTypeResolver {
	return &applicationTypeResolver{r}
}

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
type applicationTypeResolver struct{ *Resolver }
type authenticationResolver struct{ *Resolver }
type endpointResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type sourceResolver struct{ *Resolver }
