package interfaces

import (
	"context"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
)

type IDNSLookupService interface {
	RunScan(ctx context.Context, target results.Target) (results.ToolResult, error)
}

type IDNSLookupHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan results.ToolResult
}
