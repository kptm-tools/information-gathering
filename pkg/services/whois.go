package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/kptm-tools/common/common/enums"
	cmmn "github.com/kptm-tools/common/common/results"
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

func (s *WhoIsService) RunScan(ctx context.Context, target cmmn.Target) (cmmn.ToolResult, error) {
	s.Logger.Info("Running WhoIs scanner...")

	select {
	case <-ctx.Done():
		s.Logger.Warn("Context cancelled during WhoIs search", "target", target)
		return cmmn.ToolResult{}, ctx.Err()
	default:
		// Proceed with the operation
	}

	whoIsRaw, err := whois.Whois(target.Value)
	if err != nil {
		s.Logger.Error("Error fetching WHOIS", "target", target, "error", err)
		return cmmn.ToolResult{}, err
	}

	parsedResult, err := whoisparser.Parse(whoIsRaw)
	if err != nil {
		s.Logger.Error("Error parsing WHOIS data", "target", target, "error", err)
		return cmmn.ToolResult{}, err
	}

	return cmmn.ToolResult{
		Tool:      enums.ToolWhoIs,
		Success:   true,
		Result:    parsedResult,
		Timestamp: time.Now().Unix(),
	}, nil
}
