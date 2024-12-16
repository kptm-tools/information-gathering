package handlers

import (
	"fmt"

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

func (h *DNSLookupHandler) RunScan() error {
	targets := []string{"i2linked.com", "devteamdelta.org"}
	res, err := h.dnsLookupService.RunScan(targets)
	if err != nil {
		return err
	}
	fmt.Println(res.String())

	return nil
}
