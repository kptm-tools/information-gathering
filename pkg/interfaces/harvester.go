package interfaces

import (
	"context"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
)

type IHarvesterService interface {
	RunScan(ctx context.Context, targets []string) ([]results.TargetResult, error)
	HarvestEmails(ctx context.Context, target string) ([]string, error)
	HarvestSubdomains(ctx context.Context, target string) ([]string, error)
}

type IHarvesterHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan ServiceResult
}
