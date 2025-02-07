package interfaces

import (
	"github.com/kptm-tools/common/common/pkg/events"
	"github.com/kptm-tools/common/common/pkg/results/tools"
	"golang.org/x/net/context"
)

type IWhoIsService interface {
	RunScan(ctx context.Context, target string) (tools.ToolResult, error)
}

type IWhoIsHandler interface {
	RunScan(context.Context, events.ScanStartedEvent) <-chan tools.ToolResult
}
