package handlers

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/kptm-tools/common/common/events"
	cmmn "github.com/kptm-tools/common/common/results"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
)

type DNSLookupHandler struct {
	dnsLookupService interfaces.IDNSLookupService
	logger           *slog.Logger
}

var _ interfaces.IDNSLookupHandler = (*DNSLookupHandler)(nil)

func NewDNSLookupHandler(dnsLookupService interfaces.IDNSLookupService) *DNSLookupHandler {
	return &DNSLookupHandler{
		dnsLookupService: dnsLookupService,
		logger:           slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}
}

func (h *DNSLookupHandler) RunScan(event events.ScanStartedEvent) <-chan interfaces.ServiceResult {
	c := make(chan interfaces.ServiceResult)
	// 1. Parse targets from Event (targets must be domain or IP)
	targets := event.GetDomainValues()

	go func() {
		defer close(c)
		if len(targets) == 0 {
			c <- interfaces.ServiceResult{
				ScanID:      event.ScanID,
				ServiceName: cmmn.ServiceDNSLookup,
				Result:      []cmmn.TargetResult{},
				Err:         fmt.Errorf("no valid targets"),
			}
			return
		}

		results, err := h.dnsLookupService.RunScan(targets)
		if err != nil {
			h.logger.Error("error running DNS handler scan", slog.Any("error", err))
		}

		h.logger.Debug("DNSLookup Results", slog.Any("results", results))
		c <- interfaces.ServiceResult{
			ScanID:      event.ScanID,
			ServiceName: cmmn.ServiceDNSLookup,
			Result:      results,
			Err:         err,
		}
	}()

	return c
}
