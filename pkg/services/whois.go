package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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

func (s *WhoIsService) RunScan(ctx context.Context, targets []cmmn.Target) ([]cmmn.TargetResult, error) {
	s.Logger.Info("Running WhoIs scanner...")

	var (
		tResults []cmmn.TargetResult
		errs     []error
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	wg.Add(len(targets))
	for _, target := range targets {

		go func(target cmmn.Target) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				s.Logger.Warn("Context cancelled during WhoIs search", "target", target)
				return
			default:
				// Proceed with the operation
			}

			whoIsRaw, err := whois.Whois(target.Value)
			if err != nil {
				s.Logger.Error("Error fetching WHOIS, skipping to the next target. \n", "target", target, "error", err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			parsedResult, err := whoisparser.Parse(whoIsRaw)
			if err != nil {
				s.Logger.Error("Error parsing WHOIS data, skipping to the next target. \n", "target", target, "error", err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			tRes := cmmn.TargetResult{
				Target:  target,
				Results: map[enums.ServiceName]interface{}{enums.ServiceWhoIs: parsedResult},
			}
			tResults = append(tResults, tRes)
			mu.Unlock()
		}(target)
	}

	wg.Wait()

	if len(errs) > 0 {
		return tResults, fmt.Errorf("completed with errors: %v", errs)
	}

	return tResults, nil
}
