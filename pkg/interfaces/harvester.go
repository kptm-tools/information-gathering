package interfaces

import (
	"context"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
)

type IHarvesterService interface {
	RunScan(ctx context.Context, target results.Target) (results.ToolResult, error)
	HarvestEmails(ctx context.Context, domain string) ([]string, error)
	HarvestSubdomains(ctx context.Context, domain string) ([]string, error)
}

type IHarvesterHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan results.ToolResult
}
