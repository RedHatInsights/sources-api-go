package jobs

import (
	"fmt"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/dao"
)

type AsyncDestroyJob struct {
	Tenant      int64
	WaitSeconds int
	Model       string
	Id          int64
}

func (ad AsyncDestroyJob) Delay() time.Duration {
	return time.Duration(ad.WaitSeconds) * time.Second
}

func (ad AsyncDestroyJob) Arguments() map[string]interface{} {
	return map[string]interface{}{
		"wait_seconds": ad.WaitSeconds,
		"model":        ad.Model,
		"id":           ad.Id,
	}
}

func (ad AsyncDestroyJob) Name() string {
	return "AsyncDestroyJob"
}

func (ad AsyncDestroyJob) Run() error {
	switch strings.ToLower(ad.Model) {
	case "source":
		s := dao.GetSourceDao(&ad.Tenant)

		_, err := s.Delete(&ad.Id)
		if err != nil {
			return err
		}

	case "application":
		a := dao.GetApplicationDao(&ad.Tenant)
		_, err := a.Delete(&ad.Id)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid model for async destroy job: %v", ad.Model)
	}

	return nil
}
