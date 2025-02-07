package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/kptm-tools/common/common/pkg/enums"
	"github.com/kptm-tools/common/common/pkg/results/tools"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
)

type WhoIsService struct {
	Logger *slog.Logger
}

var _ interfaces.IWhoIsService = (*WhoIsService)(nil)

func NewWhoIsService() *WhoIsService {
	return &WhoIsService{
		Logger: slog.New(slog.Default().Handler()),
	}
}

func (s *WhoIsService) RunScan(ctx context.Context, domain string) (tools.ToolResult, error) {
	s.Logger.Info("Running WhoIs scanner...")

	select {
	case <-ctx.Done():
		s.Logger.Warn("Context cancelled during WhoIs search", "target", domain)
		return tools.ToolResult{}, ctx.Err()
	default:
		// Proceed with the operation
	}

	whoIsRaw, err := whois.Whois(domain)
	if err != nil {
		s.Logger.Error("Error fetching WHOIS", "target", domain, "error", err)
		return tools.ToolResult{}, err
	}

	parsedResult, err := whoisparser.Parse(whoIsRaw)
	if err != nil {
		s.Logger.Error("Error parsing WHOIS data", "target", domain, "error", err)
		return tools.ToolResult{}, err
	}

	return tools.ToolResult{
		Tool: enums.ToolWhoIs,
		Result: &tools.WhoIsResult{
			RawData: &parsedResult,
		},
		Timestamp: time.Now().UTC(),
	}, nil
}
