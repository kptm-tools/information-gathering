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

type HarvesterHandler struct {
	harvesterService interfaces.IHarvesterService
	logger           *slog.Logger
}

var _ interfaces.IHarvesterHandler = (*HarvesterHandler)(nil)

func NewHarvesterHandler(harvesterService interfaces.IHarvesterService) *HarvesterHandler {
	return &HarvesterHandler{
		harvesterService: harvesterService,
		logger:           slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (h *HarvesterHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan tools.ToolResult {
	c := make(chan tools.ToolResult)
	// 1. Parse targets from event

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("WhoIsHandler: Scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			target, err := utils.ValidateHostForTool(event.Target.Value, enums.ToolHarvester)
			if err != nil {
				errCode := utils.ClassifyValidationErrorCode(err)

				c <- tools.ToolResult{
					Tool:   enums.ToolHarvester,
					Result: &tools.HarvesterResult{},
					Err: &tools.ToolError{
						Code:    errCode,
						Message: err.Error(),
					},
					Timestamp: time.Now().UTC(),
				}
				return
			}

			result, err := h.harvesterService.RunScan(ctx, target)
			if err != nil {
				h.logger.Error("error running Harvester Handler scan", slog.Any("error", err))
				c <- tools.ToolResult{
					Tool:   enums.ToolHarvester,
					Result: &tools.HarvesterResult{},
					Err: &tools.ToolError{
						Code:    enums.ToolError,
						Message: fmt.Sprintf("error running Harvester scan: %s", err.Error()),
					},
					Timestamp: time.Now().UTC(),
				}
				return
			}
			c <- result
		}
	}()

	return c
}
