package interfaces

import (
	"context"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
)

type IDNSLookupService interface {
	RunScan(ctx context.Context, targets []results.Target) ([]results.TargetResult, error)
}

type IDNSLookupHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan results.ServiceResult
}
