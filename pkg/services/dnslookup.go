// TODO: Add validation and additional record functions

package services

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	dom "github.com/kptm-tools/information-gathering/pkg/domain"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
)

type DNSLookupService struct {
	Logger *slog.Logger
}

var _ interfaces.IDNSLookupService = (*DNSLookupService)(nil)

func NewDNSLookupService() *DNSLookupService {
	return &DNSLookupService{
		Logger: slog.New(slog.Default().Handler()),
	}
}

func (s *DNSLookupService) RunScan(targets []string) (*dom.DNSLookupEventResult, error) {

	var (
		res  dom.DNSLookupEventResult
		errs []error
		mu   sync.Mutex
		wg   sync.WaitGroup
	)

	wg.Add(len(targets))
	for _, target := range targets {
		go func(domain string) {
			defer wg.Done()

			result, err := performDNSLookup(target)
			if err != nil {
				s.Logger.Error("Error performing DNSLookup for target `%s`: %v", target, err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			res.Hosts = append(res.Hosts, *result)
			mu.Unlock()
		}(target)
	}
	wg.Wait()

	if len(errs) > 0 {
		return &res, fmt.Errorf("completed with errors: %v", errs)
	}

	return &res, nil
}

func performDNSLookup(domain string) (*dom.DNSLookupResult, error) {
	start := time.Now()

	// Retrieve A records
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}
	var aRecords []string
	for _, ip := range ips {
		if ip.To4() != nil {
			aRecords = append(aRecords, ip.String())
		}
	}

	duration := time.Since(start)
	return &dom.DNSLookupResult{
		Domain:         domain,
		ARecords:       aRecords,
		LookupDuration: duration,
		CreatedAt:      time.Now(),
	}, nil

}
