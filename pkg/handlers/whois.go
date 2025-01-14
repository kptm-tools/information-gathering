package handlers

import (
	"fmt"
	"log/slog"
	"os"

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

func (h *WhoIsHandler) RunScan(event events.ScanStartedEvent) <-chan interfaces.ServiceResult {
	c := make(chan interfaces.ServiceResult)

	go func() {
		defer close(c)
		// 1. Parse targets from StartSCAN event:
		targets := event.GetDomainValues()

		if len(targets) == 0 {
			c <- interfaces.ServiceResult{
				ScanID:      event.ScanID,
				ServiceName: cmmn.ServiceWhoIs,
				Result:      []cmmn.TargetResult{},
				Err:         fmt.Errorf("no valid targets"),
			}

		}
		results, err := h.whoIsService.RunScan(targets)
		if err != nil {
			h.logger.Error("failed to run whoIs scan", slog.Any("error", err))
		}

		h.logger.Info("WhoIs Results", slog.Any("results", results))
		c <- interfaces.ServiceResult{
			ScanID:      event.ScanID,
			ServiceName: cmmn.ServiceWhoIs,
			Result:      results,
			Err:         err,
		}
	}()

	return c
}
