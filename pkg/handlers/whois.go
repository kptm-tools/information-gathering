package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kptm-tools/common/common/enums"
	"github.com/kptm-tools/common/common/events"
	cmmn "github.com/kptm-tools/common/common/results"
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

func (h *WhoIsHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan interfaces.ServiceResult {
	c := make(chan interfaces.ServiceResult)

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("WhoIsHandler: Scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			// 1. Parse targets from StartSCAN event:
			targets := event.GetDomainValues()

			if len(targets) == 0 {
				c <- interfaces.ServiceResult{
					ScanID:      event.ScanID,
					ServiceName: enums.ServiceWhoIs,
					Result:      []cmmn.TargetResult{},
					Err:         fmt.Errorf("no valid targets"),
				}

			}
			results, err := h.whoIsService.RunScan(ctx, targets)
			if err != nil {
				h.logger.Error("failed to run whoIs scan", slog.Any("error", err))
			}

			h.logger.Info("WhoIs Results", slog.Any("results", results))
			c <- interfaces.ServiceResult{
				ScanID:      event.ScanID,
				ServiceName: enums.ServiceWhoIs,
				Result:      results,
				Err:         err,
			}
		}
	}()

	return c
}
