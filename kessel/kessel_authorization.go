package kessel

import (
	"context"
	"fmt"

	"github.com/RedHatInsights/sources-api-go/kessel/buf/gen/kessel/relations/v1beta1"
	"google.golang.org/grpc"
)

func HasPermissionOnWorkspace(ctx context.Context, resourceId string)(bool, error){
	//Kessel Connection
	conn, err := &grpc.ClientConn{} //Need to fix 
	if err != nil{
		return false, fmt.Errorf("failed to connect to Kessel service: %w", err)
	}


	client := v1beta1.NewKesselCheckServiceClient(conn)

	request := v1beta1.CheckRequest{
		Relation:    "sources_manage_all",
		Resource:    &v1beta1.ObjectReference{
			Type:  &v1beta1.ObjectType{
				Name: "workspace",
				Namespace: "rbac",
			},
			Id: "",
		},
		Subject:    &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Name: "principal",
					Namespace: "rbac",
				},
				Id: "something",
			},
		},
	}

	response, err := client.Check(ctx, &request)
	if err != nil {
		return false, fmt.Errorf("error checking the workspace permisssion: %w", err)
	}

	return true, nil
}
