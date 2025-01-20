package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kptm-tools/common/common/enums"
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

func (h *DNSLookupHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan cmmn.ServiceResult {
	c := make(chan cmmn.ServiceResult)
	// 1. Parse targets from Event (targets must be domain or IP)
	targets := event.GetDomainTargets()

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("DNSLookupHandler: scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			if len(targets) == 0 {
				c <- cmmn.ServiceResult{
					ScanID:      event.ScanID,
					ServiceName: enums.ServiceDNSLookup,
					Result:      []cmmn.TargetResult{},
					Err:         fmt.Errorf("no valid targets"),
				}
				return
			}

			results, err := h.dnsLookupService.RunScan(ctx, targets)
			if err != nil {
				h.logger.Error("error running DNS handler scan", slog.Any("error", err))
			}

			h.logger.Debug("DNSLookup Results", slog.Any("results", results))
			c <- cmmn.ServiceResult{
				ScanID:      event.ScanID,
				ServiceName: enums.ServiceDNSLookup,
				Result:      results,
				Err:         err,
			}
		}
	}()

	return c
}
