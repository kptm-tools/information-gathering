package services

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	cmmn "github.com/kptm-tools/common/common/results"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"golang.org/x/net/publicsuffix"
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

func (s *WhoIsService) RunScan(targets []string) ([]cmmn.TargetResult, error) {
	s.Logger.Info("Running WhoIs scanner...")

	var (
		tResults []cmmn.TargetResult
		errs     []error
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	wg.Add(len(targets))
	for _, target := range targets {

		go func(target string) {
			defer wg.Done()

			targetDomain, err := getDomainFromURL(target)
			if err != nil {
				s.Logger.Error("Error parsing domain from URL for target `%s`: %+v, skipping to the next target.\n", target, err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			whoIsRaw, err := whois.Whois(targetDomain)
			if err != nil {
				s.Logger.Error("Error fetching WHOIS for target `%s`: %+v, skipping to the next target. \n", targetDomain, err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			parsedResult, err := whoisparser.Parse(whoIsRaw)
			if err != nil {
				s.Logger.Error("Error parsing WHOIS data for target `%s`: %+v, skipping to the next target. \n", targetDomain, err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			tRes := cmmn.TargetResult{
				Target:  target,
				Results: map[cmmn.ServiceName]interface{}{cmmn.ServiceWhoIs: parsedResult},
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

func getDomainFromURL(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}

	hostname := parsedURL.Hostname()

	domain, err := publicsuffix.EffectiveTLDPlusOne(hostname)
	if err != nil {
		return "", fmt.Errorf("failed to get the base domain: %v", err)
	}

	// Return the domain
	return domain, nil
}
