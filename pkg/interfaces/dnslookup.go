package interfaces

import (
	"context"

	"github.com/kptm-tools/common/common/pkg/events"
	"github.com/kptm-tools/common/common/pkg/results/tools"
)

type IDNSLookupService interface {
	RunScan(ctx context.Context, target string) (tools.ToolResult, error)
}

type IDNSLookupHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan tools.ToolResult
}
