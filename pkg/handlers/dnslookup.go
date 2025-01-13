package handlers

import (
	"fmt"

	"github.com/kptm-tools/common/common/events"
	"github.com/kptm-tools/common/common/results"
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

func (h *DNSLookupHandler) RunScan(event events.ScanStartedEvent) ([]results.TargetResult, error) {
	// 1. Parse targets from Event (targets must be domain or IP)
	targets := event.GetDomainValues()

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets")
	}

	results, err := h.dnsLookupService.RunScan(targets)
	if err != nil {
		return nil, err
	}
	for _, res := range *results {
		fmt.Println(res.String())
	}

	// 2. Publish DNSLookup Event

	return *results, nil
}
