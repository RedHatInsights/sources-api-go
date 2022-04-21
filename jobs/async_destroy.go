package jobs

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/RedHatInsights/sources-api-go/kafka"
	"github.com/RedHatInsights/sources-api-go/service"
)

type AsyncDestroyJob struct {
	Headers     []kafka.Header `json:"headers"`
	Tenant      int64          `json:"tenant"`
	WaitSeconds int            `json:"wait_seconds"`
	Model       string         `json:"model"`
	Id          int64          `json:"id"`
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
		err := service.DeleteCascade(&ad.Tenant, "Source", ad.Id, ad.Headers)
		if err != nil {
			return err
		}
	case "application":
		err := service.DeleteCascade(&ad.Tenant, "Application", ad.Id, ad.Headers)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid model for async destroy job: %v", ad.Model)
	}

	return nil
}

func (ad AsyncDestroyJob) ToJSON() []byte {
	bytes, err := json.Marshal(&ad)
	if err != nil {
		panic(err)
	}
	return bytes
}
