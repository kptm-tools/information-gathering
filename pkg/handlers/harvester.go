package handlers

import (
	"fmt"

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

func (h *HarvesterHandler) RunScan() error {

	targets := []string{"aynitech.com"}
	results, err := h.harvesterService.RunScan(targets)
	if err != nil {
		return err
	}

	for _, res := range *results {
		fmt.Println(res.String())
	}

	return nil
}
