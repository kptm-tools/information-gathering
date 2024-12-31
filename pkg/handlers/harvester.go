package handlers

import (
	"fmt"

	"github.com/kptm-tools/common/common/events"
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

func (h *HarvesterHandler) RunScan(event events.ScanStartedEvent) error {

	// 1. Parse targets from event
	targets := event.Targets
	results, err := h.harvesterService.RunScan(targets)
	if err != nil {
		return err
	}

	for _, res := range results {
		fmt.Println(res.String())
	}

	// 2. Publish HarvesterEvent to bus

	return nil
}
