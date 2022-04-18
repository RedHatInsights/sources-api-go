package jobs

import (
	"fmt"
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
	l "github.com/RedHatInsights/sources-api-go/logger"
	m "github.com/RedHatInsights/sources-api-go/model"
	"github.com/RedHatInsights/sources-api-go/service"
	"github.com/RedHatInsights/sources-api-go/util"
)

type SuperkeyDestroyJob struct {
	Identity string
	Tenant   int64
	Model    string
	Id       int64
}

func (sk SuperkeyDestroyJob) Delay() time.Duration {
	// run this job immediately, no delay.
	return 0
}

func (sk SuperkeyDestroyJob) Arguments() map[string]interface{} {
	return map[string]interface{}{
		"model": sk.Model,
		"id":    sk.Id,
	}
}

func (sk SuperkeyDestroyJob) Name() string {
	return "SuperkeyDestroyJob"
}

func (sk SuperkeyDestroyJob) Run() error {
	l.Log.Infof("Running [%v] with [%v]", sk.Name(), sk.Arguments())

	switch sk.Model {
	case "source":
		return sk.sendForSource(sk.Id)
	case "application":
		return sk.sendForApplication(sk.Id)
	default:
		return fmt.Errorf("unsupported model for superkey: %v", sk.Model)
	}
}

// Lists all applications for a source, sends the destroy request for each, then
// enqueues deletion of itself.

// the sub-jobs enqueue deletion of their respective resources.
func (sk SuperkeyDestroyJob) sendForSource(id int64) error {
	l.Log.Infof("Sending SuperKey Delete request for source %v", sk.Id)

	a := dao.GetApplicationDao(&sk.Tenant)

	apps, _, err := a.ListForSource(&id, 100, 0, []util.Filter{})
	if err != nil {
		return fmt.Errorf("failed to list applications for source: %v", err)
	}

	errors := make([]error, 0)
	for i := range apps {
		err := sk.sendForApplication(apps[i].ID)
		if err != nil {
			l.Log.Warnf("Error sending destroy request for application %v: %v", apps[i].ID, err)
			errors = append(errors, err)
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf("ran into errors sending delete requests for application: %v", errors)
	}

	// destroy the source after waiting 15 seconds
	Enqueue(&AsyncDestroyJob{
		Tenant:      sk.Tenant,
		WaitSeconds: 15,
		Model:       "source",
		Id:          id,
	})

	return nil
}

func (sk SuperkeyDestroyJob) sendForApplication(id int64) error {
	l.Log.Infof("Sending SuperKey Delete request for application %v", sk.Id)

	err := service.SendSuperKeyDeleteRequest(sk.Identity, &m.Application{ID: id, TenantID: sk.Tenant})
	if err != nil {
		return err
	}

	// destroy the application after waiting 15 seconds
	Enqueue(&AsyncDestroyJob{
		Tenant:      sk.Tenant,
		WaitSeconds: 15,
		Model:       "application",
		Id:          id,
	})

	return nil
}
