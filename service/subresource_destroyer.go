package service

import (
	"fmt"

	"github.com/RedHatInsights/sources-api-go/dao"
	"github.com/RedHatInsights/sources-api-go/kafka"
	logging "github.com/RedHatInsights/sources-api-go/logger"
	"github.com/RedHatInsights/sources-api-go/model"
)

// DeleteCascade removes the resource type and all its dependants, raising an event for every deleted resource. Returns
// an error when the resources and its dependants could not be successfully removed.
//
// - In the case of the resource being a "Source", it removes its applications, its endpoints and its Red Hat Connector
// connections.
// - In the other cases, it removes the resource itself.
//
// In both cases the authentications are fetched for every single resource and sub resource, and they get deleted in
// batch.
//
// In the case of not being able to delete authentications, we simply log the error and leave them dangling, since we
// might either have a database or Vault backing our authentications, and we might need manual action to remove the
// ones that were left undeleted. In the case of a database it is true that it could be wrapped in a transaction, but
// with Vault that is not possible. What if we start deleting authentications and then some of them error out? We would
// be able to recover the database data, but some of their authentications would be gone, therefore leaving the
// resource in an inconsistent state.
//
// Plus, it's not the client's fault that we weren't able to delete the authentications, so by taking this approach
// the clients don't keep an "unremovable" resource because of some failure in deleting the authentications.
//
// Finally, those authentications are safely encrypted so they can stay in their datastores until we manually remove
// them.
func DeleteCascade(tenantId *int64, resourceType string, resourceId int64, headers []kafka.Header) error {
	authenticationsDao := dao.GetAuthenticationDao(tenantId)
	var authentications []model.Authentication

	switch resourceType {
	case "Source":
		sourceDao := dao.GetSourceDao(tenantId)
		applicationAuthentications, applications, endpoints, rhcConnections, source, err := sourceDao.DeleteCascade(resourceId)
		if err != nil {
			return fmt.Errorf(`could not completely delete the source: %s`, err)
		}

		// Raise an event for the deleted application authentications.
		for _, appAuth := range applicationAuthentications {
			err = RaiseEvent("ApplicationAuthentication.destroy", &appAuth, headers)
			if err != nil {
				logging.Log.Errorf(`Event "ApplicationAuthentication.destroy" could not be raised for application authentication %v: %s`, appAuth.ToEvent(), err)
			}
		}

		// Raise events for the deleted applications.
		var appIds []int64
		for _, app := range applications {
			appIds = append(appIds, app.ID)
			err := RaiseEvent("Application.destroy", &app, headers)
			if err != nil {
				logging.Log.Errorf(`Event "Application.destroy" could not be raised for application %v: %s`, app.ToEvent(), err)
			}
		}

		// Raise events for the deleted endpoints.
		var endpointIds []int64
		for _, endpoint := range endpoints {
			endpointIds = append(endpointIds, endpoint.ID)
			err := RaiseEvent("Endpoint.destroy", &endpoint, headers)
			if err != nil {
				logging.Log.Errorf(`Event "Endpoint.destroy" could not be raised for endpoint %v: %s`, endpoint.ToEvent(), err)
			}
		}

		// Raise events for the deleted connections.
		for _, connection := range rhcConnections {
			err := RaiseEvent("RhcConnection.destroy", &connection, headers)
			if err != nil {
				logging.Log.Errorf(`Event "RhcConnection.destroy" could not be raised for rhcConnection %v: %s`, connection.ToEvent(), err)
			}
		}

		// Raise an event for the source itself.
		err = RaiseEvent("Source.destroy", source, headers)
		if err != nil {
			logging.Log.Errorf(`Event "Source.destroy" could not be raised for source %v: %s`, source.ToEvent(), err)
		}

		// Fetch all the authentications from the resources.
		resourceTypes := []struct {
			ResourceType string
			ResourceIds  []int64
		}{
			{ResourceType: "Application", ResourceIds: appIds},
			{ResourceType: "Endpoint", ResourceIds: endpointIds},
			{ResourceType: "Source", ResourceIds: []int64{resourceId}},
		}

		for _, res := range resourceTypes {
			auths, err := authenticationsDao.ListIdsForResource(res.ResourceType, res.ResourceIds)
			if err != nil {
				logging.Log.Errorf(`[resource_type: "%s"][resource_ids: "%v"] Could not fetch authentications: %s`, res.ResourceType, res.ResourceIds, err)
			}

			authentications = append(authentications, auths...)
		}

	case "Application":
		applicationsDao := dao.GetApplicationDao(tenantId)
		applicationAuthentications, application, err := applicationsDao.DeleteCascade(resourceId)
		if err != nil {
			return fmt.Errorf(`could not completely delete the application: %s`, err)
		}

		// Raise an event for the deleted application authentications.
		for _, appAuth := range applicationAuthentications {
			err = RaiseEvent("ApplicationAuthentication.destroy", &appAuth, headers)
			if err != nil {
				logging.Log.Errorf(`Event "ApplicationAuthentication.destroy" could not be raised for application authentication %v: %s`, appAuth.ToEvent(), err)
			}
		}

		// Raise an event for the deleted application itself.
		err = RaiseEvent("Application.destroy", application, headers)
		if err != nil {
			logging.Log.Errorf(`Event "Application.destroy" could not be raised for application %v: %s`, application.ToEvent(), err)
		}

		// Fetch all the application's authentications.
		auths, err := authenticationsDao.ListIdsForResource("Application", []int64{resourceId})
		if err != nil {
			logging.Log.Errorf(`[resource_type: "Application"][resource_id: "%v"] Could not fetch authentications: %s`, resourceId, err)
		}

		authentications = append(authentications, auths...)
	case "Endpoint":
		// Delete the endpoint.
		endpointDao := dao.GetEndpointDao(tenantId)
		endpoint, err := endpointDao.Delete(&resourceId)
		if err != nil {
			return fmt.Errorf(`could not delete the endpoint: %s`, err)
		}

		// Raise an event for the deleted endpoint.
		err = RaiseEvent("Endpoint.destroy", endpoint, headers)
		if err != nil {
			logging.Log.Errorf(`Event "Endpoint.destroy" could not be raised for endpoint %v: %s`, endpoint.ToEvent(), err)
		}

		// Fetch all the endpoint's authentications.
		auths, err := authenticationsDao.ListIdsForResource("Endpoint", []int64{resourceId})
		if err != nil {
			logging.Log.Errorf(`[resource_type: "Endpoint"][resource_id: "%v"] Could not fetch authentications: %s`, resourceId, err)
		}

		authentications = append(authentications, auths...)
	}

	// Delete all the authentications.
	deletedAuths, err := authenticationsDao.BulkDelete(authentications)
	if err != nil {
		logging.Log.Errorf(`Could not delete authentications: %s`, err)
	}

	for _, deletedAuth := range deletedAuths {
		// Raise an event for the deleted authentication.
		err = RaiseEvent("Authentication.destroy", &deletedAuth, headers)
		if err != nil {
			logging.Log.Errorf(`Could not raise "Authentication.destroy" event for authentication %v`, err)
		}
	}

	return nil
}
