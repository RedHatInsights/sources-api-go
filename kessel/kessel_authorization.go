package kessel

import (
	"context"
	"log"

	"github.com/RedHatInsights/sources-api-go/kessel/buf/gen/kessel/relations/v1beta1"
	"google.golang.org/grpc"
)

func HasPermissionOnWorkspace(ctx context.Context, resourceId string){
	client := v1beta1.NewKesselCheckServiceClient(&grpc.ClientConn{})

	request := v1beta1.CheckRequest{
		Relation:    "sources_manage_all",
		Resource:    &v1beta1.ObjectReference{
			Type:  &v1beta1.ObjectType{
				Name: "rbac/workspace",
			},
			Id: "",
		},
		Subject:    &v1beta1.SubjectReference{
			Subject: &v1beta1.ObjectReference{
				Type: &v1beta1.ObjectType{
					Name: "rbac/principal",
					Namespace: "redhat",
				},
				Id: "something",
			},
		},
	}

	response, err := client.Check(ctx, &request)
	if err != nil {
		log.Printf("Error checking permission %v", err)
		return
	}

	log.Printf("Permission check result: %v",response)
}