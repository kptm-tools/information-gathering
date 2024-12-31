package interfaces

import (
	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
)

type IHarvesterService interface {
	RunScan(targets []string) ([]results.TargetResult, error)
	HarvestEmails(target string) ([]string, error)
	HarvestSubdomains(target string) ([]string, error)
}

type IHarvesterHandler interface {
	RunScan(events.ScanStartedEvent) error
}
