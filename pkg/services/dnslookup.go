package services

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	cmmn "github.com/kptm-tools/common/common/results"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"github.com/miekg/dns"
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

func (s *DNSLookupService) RunScan(targets []string) ([]cmmn.TargetResult, error) {

	var (
		targetResults []cmmn.TargetResult
		errs          []error
		mu            sync.Mutex
		wg            sync.WaitGroup
	)

	wg.Add(len(targets))
	for _, target := range targets {
		if !isValidDomain(target) {
			s.Logger.Error("Not a valid domain", "domain", target)
			continue
		}

		go func(domain string) {
			defer wg.Done()

			result, err := performDNSLookup(target)
			if err != nil {
				s.Logger.Error("Error performing DNSLookup for target ", "target", target, "error", err)
				mu.Lock()
				errs = append(errs, err...)
				mu.Unlock()
				return
			}

			tResult := cmmn.TargetResult{
				Target:  domain,
				Results: map[cmmn.ServiceName]interface{}{cmmn.ServiceHarvester: result},
			}

			mu.Lock()
			targetResults = append(targetResults, tResult)
			mu.Unlock()
		}(target)
	}
	wg.Wait()

	if len(errs) > 0 {
		var formattedErrors []string
		for _, e := range errs {
			formattedErrors = append(formattedErrors, e.Error())
		}
		return targetResults, fmt.Errorf("completed with errors:\n%s", strings.Join(formattedErrors, "\n"))
	}

	return targetResults, nil
}

func performDNSLookup(domain string) (*cmmn.DNSLookupResult, []error) {

	var (
		records       []cmmn.DNSRecord
		DNSSECEnabled bool
		errs          []error
	)
	start := time.Now()
	wantRecords := []uint16{
		dns.TypeA,
		dns.TypeAAAA,
		dns.TypeCNAME,
		dns.TypeTXT,
		dns.TypeNS,
		dns.TypeMX,
		dns.TypeSOA,
		dns.TypeDNSKEY,
	}

	for _, recordType := range wantRecords {
		typeRecords, err := QueryDNSRecord(domain, recordType)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		records = append(records, typeRecords...)
	}

	// Check if we got a DNSKeyRecord somewhere
	if cmmn.HasDNSKeyRecord(records) {
		DNSSECEnabled = true
	}

	duration := time.Since(start)

	return &cmmn.DNSLookupResult{
		Domain:         domain,
		DNSRecords:     records,
		DNSSECEnabled:  DNSSECEnabled,
		LookupDuration: duration,
		CreatedAt:      time.Now(),
	}, errs

}

// QueryDNSRecord fetches available records of the specified type and returns TTL information
func QueryDNSRecord(domain string, recordType uint16) ([]cmmn.DNSRecord, error) {
	var records []cmmn.DNSRecord

	r := cmmn.GoogleResolver
	// Create DNS message
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), recordType)

	// Use a DNS resolver
	c := new(dns.Client)
	res, _, err := c.Exchange(m, r)
	if err != nil {
		return nil, fmt.Errorf("failed to query type `%v` records for domain %s: %w", recordType, domain, err)
	}

	// Parse the answers
	for _, answer := range res.Answer {
		switch record := answer.(type) {
		case *dns.A:
			records = append(records, cmmn.DNSRecord{
				Name:  record.Header().Name,
				Type:  cmmn.ARecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.A.String(),
			})
		case *dns.AAAA:
			records = append(records, cmmn.DNSRecord{
				Name:  record.Header().Name,
				Type:  cmmn.AAAARecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.AAAA.String(),
			})
		case *dns.CNAME:
			records = append(records, cmmn.DNSRecord{
				Name:  record.Header().Name,
				Type:  cmmn.CNAMERecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Target,
			})
		case *dns.MX:
			records = append(records, cmmn.DNSRecord{
				Name: record.Hdr.Name,
				Type: cmmn.MXRecord,
				TTL:  int(record.Hdr.Ttl),
				Value: cmmn.MailExchange{
					Host:     record.Mx,
					Priority: int(record.Preference),
				},
			})
		case *dns.TXT:
			records = append(records, cmmn.DNSRecord{
				Name:  record.Hdr.Name,
				Type:  cmmn.TXTRecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Txt,
			})
		case *dns.NS:
			records = append(records, cmmn.DNSRecord{
				Name:  record.Hdr.Name,
				Type:  cmmn.NSRecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Ns,
			})
		case *dns.SOA:
			records = append(records, cmmn.DNSRecord{
				Name: record.Hdr.Name,
				Type: cmmn.SOARecord,
				TTL:  int(record.Hdr.Ttl),
				Value: cmmn.StartOfAuthority{
					PrimaryNS:  record.Ns,
					AdminEmail: record.Mbox,
					Serial:     int(record.Serial),
					Refresh:    int(record.Refresh),
					Retry:      int(record.Retry),
					Expire:     int(record.Expire),
					MinimumTTL: int(record.Minttl),
				},
			})
		case *dns.DNSKEY:
			records = append(records, cmmn.DNSRecord{
				Name: record.Hdr.Name,
				Type: cmmn.DNSKeyRecord,
				TTL:  int(record.Hdr.Ttl),
				Value: cmmn.DNSKey{
					Flags:     int(record.Flags),
					Protocol:  int(record.Protocol),
					Algorithm: int(record.Algorithm),
				},
			})
		}
	}
	return records, nil
}

func isValidDomain(domain string) bool {
	// net.LookupHost validates the domain and resolves it
	if _, err := net.LookupHost(domain); err != nil {
		return false
	}
	return true
}
