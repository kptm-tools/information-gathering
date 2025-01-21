package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kptm-tools/common/common/enums"
	"github.com/kptm-tools/common/common/events"
	cmmn "github.com/kptm-tools/common/common/results"
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

func (h *HarvesterHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan cmmn.ToolResult {

	c := make(chan cmmn.ToolResult)
	// 1. Parse targets from event

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("WhoIsHandler: Scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			if !event.HasDomainTarget() {
				c <- cmmn.ToolResult{
					Tool:      enums.ToolHarvester,
					Success:   false,
					Err:       fmt.Errorf("no valid targets"),
					Timestamp: time.Now().Unix(),
				}
			}

			result, err := h.harvesterService.RunScan(ctx, event.Target)
			if err != nil {
				h.logger.Error("error running Harvester Handler scan", slog.Any("error", err))
			}
			c <- result
		}

	}()

	return c
}
