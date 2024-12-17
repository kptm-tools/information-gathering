// TODO: Add validation and additional record functions

package services

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	dom "github.com/kptm-tools/information-gathering/pkg/domain"
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
				s.Logger.Error("Error performing DNSLookup for target ", target, err)
				mu.Lock()
				errs = append(errs, err...)
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
		var formattedErrors []string
		for _, e := range errs {
			formattedErrors = append(formattedErrors, e.Error())
		}
		return &res, fmt.Errorf("completed with errors:\n%s", errs)
	}

	return &res, nil
}

func performDNSLookup(domain string) (*dom.DNSLookupResult, []error) {

	var (
		records       []dom.DNSRecord
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
	if hasDNSKeyRecord(records) {
		DNSSECEnabled = true
	}

	duration := time.Since(start)

	return &dom.DNSLookupResult{
		Domain:         domain,
		DNSRecords:     records,
		DNSSECEnabled:  DNSSECEnabled,
		LookupDuration: duration,
		CreatedAt:      time.Now(),
	}, errs

}

// QueryDNSRecord fetches available records of the specified type and returns TTL information
func QueryDNSRecord(domain string, recordType uint16) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord

	r := dom.GoogleResolver
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
			records = append(records, dom.DNSRecord{
				Name:  record.Header().Name,
				Type:  dom.ARecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.A.String(),
			})
		case *dns.AAAA:
			records = append(records, dom.DNSRecord{
				Name:  record.Header().Name,
				Type:  dom.AAAARecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.AAAA.String(),
			})
		case *dns.CNAME:
			records = append(records, dom.DNSRecord{
				Name:  record.Header().Name,
				Type:  dom.CNAMERecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Target,
			})
		case *dns.MX:
			records = append(records, dom.DNSRecord{
				Name: record.Hdr.Name,
				Type: dom.MXRecord,
				TTL:  int(record.Hdr.Ttl),
				Value: dom.MailExchange{
					Host:     record.Mx,
					Priority: int(record.Preference),
				},
			})
		case *dns.TXT:
			records = append(records, dom.DNSRecord{
				Name:  record.Hdr.Name,
				Type:  dom.TXTRecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Txt,
			})
		case *dns.NS:
			records = append(records, dom.DNSRecord{
				Name:  record.Hdr.Name,
				Type:  dom.NSRecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Ns,
			})
		case *dns.SOA:
			records = append(records, dom.DNSRecord{
				Name: record.Hdr.Name,
				Type: dom.SOARecord,
				TTL:  int(record.Hdr.Ttl),
				Value: dom.StartOfAuthority{
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
			records = append(records, dom.DNSRecord{
				Name: record.Hdr.Name,
				Type: dom.DNSKeyRecord,
				TTL:  int(record.Hdr.Ttl),
				Value: dom.DNSKey{
					Flags:     int(record.Flags),
					Protocol:  int(record.Protocol),
					Algorithm: int(record.Algorithm),
				},
			})
		}
	}
	return records, nil
}

func hasDNSKeyRecord(records []dom.DNSRecord) bool {
	for _, record := range records {
		if record.Type == dom.DNSKeyRecord {
			return true
		}
	}
	return false
}
