package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

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

func (h *DNSLookupHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan cmmn.ToolResult {
	c := make(chan cmmn.ToolResult)

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("DNSLookupHandler: scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			if !event.HasDomainTarget() {
				c <- cmmn.ToolResult{
					Tool: enums.ToolDNSLookup,
					Err: &cmmn.ToolError{
						Code:    enums.ValidationError,
						Message: fmt.Sprintf("invalid target: %s", event.Target.Value),
					},
					Timestamp: time.Now().UTC(),
				}
				return
			}

			result, err := h.dnsLookupService.RunScan(ctx, event.Target)
			if err != nil {
				h.logger.Error("error running DNS handler scan", slog.Any("error", err))
				c <- cmmn.ToolResult{
					Tool: enums.ToolDNSLookup,
					Err: &cmmn.ToolError{
						Code:    enums.ToolError,
						Message: fmt.Sprintf("error running DNS handler: %s", err.Error()),
					},
				}
				return
			}

			h.logger.Debug("DNSLookup Results", slog.Any("results", result))
			c <- result
		}
	}()

	return c
}
