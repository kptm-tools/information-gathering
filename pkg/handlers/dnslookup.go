package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kptm-tools/common/common/pkg/customerrors"
	"github.com/kptm-tools/common/common/pkg/enums"
	"github.com/kptm-tools/common/common/pkg/events"
	"github.com/kptm-tools/common/common/pkg/results/tools"
	"github.com/kptm-tools/common/common/pkg/utils"
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

func (h *DNSLookupHandler) RunScan(ctx context.Context, event events.ScanStartedEvent) <-chan tools.ToolResult {
	c := make(chan tools.ToolResult)

	go func() {
		defer close(c)

		select {
		case <-ctx.Done():
			h.logger.Info("DNSLookupHandler: scan cancelled", slog.Any("scanID", event.ScanID))
			return
		default:
			target, err := utils.ValidateHostForTool(event.Target.Value, enums.ToolDNSLookup)
			if err != nil {
				var errIncompatibleTool *customerrors.ToolIncompatibleError
				var errCode enums.ErrorCode
				if errors.As(err, &errIncompatibleTool) {
					errCode = enums.ToolSkippedError
				} else {
					errCode = enums.ValidationError
				}

				c <- tools.ToolResult{
					Tool:   enums.ToolDNSLookup,
					Result: &tools.DNSLookupResult{},
					Err: &tools.ToolError{
						Code:    errCode,
						Message: err.Error(),
					},
					Timestamp: time.Now().UTC(),
				}
				return
			}

			result, err := h.dnsLookupService.RunScan(ctx, target)
			if err != nil {
				h.logger.Error("error running DNS handler scan", slog.Any("error", err))
				c <- tools.ToolResult{
					Tool:   enums.ToolDNSLookup,
					Result: &tools.DNSLookupResult{},
					Err: &tools.ToolError{
						Code:    enums.ToolError,
						Message: fmt.Sprintf("error running DNS handler: %s", err.Error()),
					},
					Timestamp: time.Now().UTC(),
				}
				return
			}

			h.logger.Debug("DNSLookup Results", slog.Any("results", result))
			c <- result
		}
	}()

	return c
}
