package interfaces

import (
	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
)

type IDNSLookupService interface {
	RunScan(targets []string) (*[]results.TargetResult, error)
}

type IDNSLookupHandler interface {
	RunScan(events.ScanStartedEvent) ([]results.TargetResult, error)
}
