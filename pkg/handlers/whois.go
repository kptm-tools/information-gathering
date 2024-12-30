package handlers

import (
	"fmt"

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

func (h *WhoIsHandler) RunScan() error {
	// Parse targets from StartSCAN event:
	targets := []string{"whois.verisign-grs.com", "i2linked.com", "twitterapp.devteamdelta.org", "thissubdomaindoesnotexist.devteamdelta.org"}
	results, err := h.whoIsService.RunScan(targets)

	if err != nil {
		return err
	}
	for _, res := range *results {
		fmt.Println(res.String())
	}
	return nil
}
