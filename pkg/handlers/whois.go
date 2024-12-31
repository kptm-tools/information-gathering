package handlers

import (
	"fmt"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
)

type WhoIsHandler struct {
	whoIsService interfaces.IWhoIsService
}

var _ interfaces.IWhoIsHandler = (*WhoIsHandler)(nil)

func NewWhoIsHandler(whoIsService interfaces.IWhoIsService) *WhoIsHandler {
	return &WhoIsHandler{
		whoIsService: whoIsService,
	}
}

func (h *WhoIsHandler) RunScan(event events.ScanStartedEvent) error {
	// 1. Parse targets from StartSCAN event:
	targets := event.Targets
	results, err := h.whoIsService.RunScan(targets)

	if err != nil {
		return err
	}
	for _, res := range *results {
		fmt.Println(res.String())
	}

	// 2. Publish WhoIs Event to bus
	return nil
}
