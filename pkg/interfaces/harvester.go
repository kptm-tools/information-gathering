package interfaces

import (
	"context"

	"github.com/kptm-tools/common/common/pkg/events"
	"github.com/kptm-tools/common/common/pkg/results/tools"
)

type IHarvesterService interface {
	RunScan(ctx context.Context, target string) (tools.ToolResult, error)
	HarvestEmails(ctx context.Context, domain string) ([]string, error)
	HarvestSubdomains(ctx context.Context, domain string) ([]string, error)
}

type IHarvesterHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan tools.ToolResult
}
