package handlers

import (
	"fmt"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
)

type HarvesterHandler struct {
	harvesterService interfaces.IHarvesterService
}

var _ interfaces.IHarvesterHandler = (*HarvesterHandler)(nil)

func NewHarvesterHandler(harvesterService interfaces.IHarvesterService) *HarvesterHandler {
	return &HarvesterHandler{
		harvesterService: harvesterService,
	}
}

func (h *HarvesterHandler) RunScan(event events.ScanStartedEvent) ([]results.TargetResult, error) {

	// 1. Parse targets from event
	targets := event.GetDomainValues()

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets")
	}

	results, err := h.harvesterService.RunScan(targets)
	if err != nil {
		return nil, err
	}

	for _, res := range results {
		fmt.Println(res.String())
	}

	// 2. Publish HarvesterEvent to bus

	return results, nil
}
