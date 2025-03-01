package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kptm-tools/common/common/pkg/enums"
	"github.com/kptm-tools/common/common/pkg/events"
	"github.com/kptm-tools/common/common/pkg/results/tools"
	"github.com/kptm-tools/common/common/pkg/utils"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
)

type WhoIsHandler struct {
	whoIsService interfaces.IWhoIsService
	logger       *slog.Logger
}

var _ interfaces.IWhoIsHandler = (*WhoIsHandler)(nil)

func NewWhoIsHandler(whoIsService interfaces.IWhoIsService) *WhoIsHandler {
	return &WhoIsHandler{
		whoIsService: whoIsService,
		logger:       slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (h *WhoIsHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan tools.ToolResult {
	c := make(chan tools.ToolResult)

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("WhoIsHandler: Scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			target, err := utils.ValidateHostForTool(event.Target.Value, enums.ToolWhoIs)
			if err != nil {
				c <- tools.ToolResult{
					Tool:   enums.ToolWhoIs,
					Result: &tools.WhoIsResult{},
					Err: &tools.ToolError{
						Code:    enums.ValidationError,
						Message: fmt.Sprintf("invalid target: %s", event.Target.Value),
					},
					Timestamp: time.Now().UTC(),
				}
				return
			}

			result, err := h.whoIsService.RunScan(ctx, target)
			if err != nil {
				h.logger.Error("failed to run whoIs scan", slog.Any("error", err))
				c <- tools.ToolResult{
					Tool:   enums.ToolWhoIs,
					Result: &tools.WhoIsResult{},
					Err: &tools.ToolError{
						Code:    enums.ToolError,
						Message: fmt.Sprintf("failed to run whoIs scan: %s", err.Error()),
					},
				}
				return
			}

			h.logger.Info("WhoIs Results", slog.Any("results", result))
			c <- result
		}
	}()

	return c
}
