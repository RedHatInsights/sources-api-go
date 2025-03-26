package kessel

import (
	"context"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/kessel/buf/gen/kessel/relations/v1beta1"
	"google.golang.org/grpc"
)

type KesselAuthorizationService interface{
	HasPermissionOnWorkspace(ctx context.Context, workspaceId string, userId string)(bool, error)
}

type kesselAuthorizationServiceImpl struct{
	kesselCheckClient v1beta1.KesselCheckServiceClient
}

func NewKesselAuthorizationService(clientConnection grpc.ClientConnInterface)(KesselAuthorizationService){
	return kesselAuthorizationServiceImpl{
		kesselCheckClient: v1beta1.NewKesselCheckServiceClient(clientConnection),
	}
}

func(k kesselAuthorizationServiceImpl)HasPermissionOnWorkspace(ctx context.Context, workspaceId string, userId string)(bool, error){

	request := v1beta1.CheckRequest{
		Relation:    "sources_manage_all",
		Resource:    &v1beta1.ObjectReference{
			Type:  	 &v1beta1.ObjectType{
				Name: "workspace",
				Namespace: "rbac",
			},
			Id: workspaceId,
		},
		Subject:    &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Name: "principal",
					Namespace: "rbac",
				},
				Id: "redhat/"+userId,
			},
		},
	}

	response, err := k.kesselCheckClient.Check(ctx, &request)
	if err != nil {
		return false, fmt.Errorf("error checking the workspace permisssion: %w", err)
	}

	return response.GetAllowed() == v1beta1.CheckResponse_ALLOWED_TRUE, nil
}
