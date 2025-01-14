package handlers

import (
	"fmt"
	"log/slog"
	"os"

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

func (h *HarvesterHandler) RunScan(event events.ScanStartedEvent) <-chan interfaces.ServiceResult {

	c := make(chan interfaces.ServiceResult)
	// 1. Parse targets from event
	targets := event.GetDomainValues()

	go func() {
		defer close(c)
		if len(targets) == 0 {
			c <- interfaces.ServiceResult{
				ScanID:      event.ScanID,
				ServiceName: cmmn.ServiceHarvester,
				Result:      []cmmn.TargetResult{},
				Err:         fmt.Errorf("no valid targets"),
			}
		}

		results, err := h.harvesterService.RunScan(targets)
		if err != nil {
			h.logger.Error("error running Harvester Handler scan", slog.Any("error", err))
		}
		h.logger.Info("Harvester Results", slog.Any("results", results))
		c <- interfaces.ServiceResult{
			ScanID:      event.ScanID,
			ServiceName: cmmn.ServiceHarvester,
			Result:      results,
			Err:         err,
		}
	}()

	return c
}
