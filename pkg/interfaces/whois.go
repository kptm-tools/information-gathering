package interfaces

import (
	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
	"golang.org/x/net/context"
)

type IWhoIsService interface {
	RunScan(ctx context.Context, targets []results.Target) ([]results.TargetResult, error)
}

type IWhoIsHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan ServiceResult
}
