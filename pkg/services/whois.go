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
	maxRetries int
	retryDelay time.Duration
}

type WhoIsServiceOptions struct {
	MaxRetries int
	RetryDelay time.Duration
}

var _ interfaces.IWhoIsService = (*WhoIsService)(nil)

func NewWhoIsService(opts *WhoIsServiceOptions) *WhoIsService {
	if opts == nil {
		opts = &WhoIsServiceOptions{}
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	retryDelay := opts.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	return &WhoIsService{
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

func (s *WhoIsService) RunScan(ctx context.Context, domain string) (tools.ToolResult, error) {
	slog.Info("Running WhoIs scanner...")

	select {
	case <-ctx.Done():
		slog.Warn("Context cancelled during WhoIs search", "target", domain)
		return tools.ToolResult{}, ctx.Err()
	default:
		// Proceed with the operation
	}

	var whoIsRaw string
	var err error

	for retryCount := 0; retryCount < s.maxRetries; retryCount++ {
		if retryCount > 0 { // Log only on retries, not on the initial attempt
			newRetryDelay := s.calculateRetryDelay(retryCount)
			slog.Warn("WhoIs request failed, retrying",
				slog.Int("attempt", retryCount+1),
				slog.Any("whois_error", err),
				slog.Duration("delay", newRetryDelay))
			time.Sleep(newRetryDelay)
		}
		whoIsRaw, err = whois.Whois(domain)
		if err == nil {
			break
		}
	}

	parsedResult, err := whoisparser.Parse(whoIsRaw)
	if err != nil {
		slog.Error("Error parsing WHOIS data", "target", domain, "error", err)
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

func (s *WhoIsService) calculateRetryDelay(attempt int) time.Duration {
	// Exponential backoff with jitter
	delay := s.retryDelay * time.Duration(1<<uint(attempt))
	jitter := time.Duration(int64(float64(delay) * 0.2)) // +/- 20% jitter
	delay += jitter

	if delay > 15*time.Second {
		delay = 15 * time.Second
	}
	return delay
}
