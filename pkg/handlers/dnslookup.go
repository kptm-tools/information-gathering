package handlers

import (
	"fmt"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
)

type DNSLookupHandler struct {
	dnsLookupService interfaces.IDNSLookupService
}

var _ interfaces.IDNSLookupHandler = (*DNSLookupHandler)(nil)

func NewDNSLookupHandler(dnsLookupService interfaces.IDNSLookupService) *DNSLookupHandler {
	return &DNSLookupHandler{
		dnsLookupService: dnsLookupService,
	}
}

func (h *DNSLookupHandler) RunScan(event events.ScanStartedEvent) error {
	// 1. Parse targets from Event
	targets := event.Targets
	results, err := h.dnsLookupService.RunScan(targets)
	if err != nil {
		return err
	}
	for _, res := range *results {
		fmt.Println(res.String())
	}

	// 2. Publish DNSLookup Event

	return nil
}
