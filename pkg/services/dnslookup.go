package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kptm-tools/common/common/pkg/enums"
	"github.com/kptm-tools/common/common/pkg/results/tools"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"github.com/miekg/dns"
)

type DNSLookupService struct {
	maxRetries int
	retryDelay time.Duration
}

type DNSLookupOpts struct {
	MaxRetries int
	RetryDelay time.Duration
}

var _ interfaces.IDNSLookupService = (*DNSLookupService)(nil)

func NewDNSLookupService(opts *DNSLookupOpts) *DNSLookupService {
	if opts == nil {
		opts = &DNSLookupOpts{}
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	retryDelay := opts.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	return &DNSLookupService{
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

func (s *DNSLookupService) RunScan(ctx context.Context, domain string) (tools.ToolResult, error) {
	result := tools.ToolResult{
		Tool:      enums.ToolDNSLookup,
		Result:    &tools.DNSLookupResult{},
		Timestamp: time.Now().UTC(),
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		slog.Warn("Context canceled during DNS lookup", "domain", domain)
		return tools.ToolResult{}, ctx.Err()
	default:
		// Proceed with the operation
	}

	var lookupResult tools.DNSLookupResult
	var err error

	for retryCount := 0; retryCount < s.maxRetries; retryCount++ {
		if retryCount > 0 { // Log only retries on initial attempt
			newRetryDelay := s.calculateRetryDelay(retryCount)
			slog.Warn("DNSLookup request failed, retrying",
				slog.Int("attempt", retryCount+1),
				slog.Any("dns_error", err),
				slog.Duration("new_delay", newRetryDelay))
			time.Sleep(newRetryDelay)
		}
		lookupResult, err = performDNSLookup(ctx, domain)
		if err == nil {
			break
		}
	}
	// If err is not nil after the retry loop
	if err != nil {
		slog.Error("Error performing DNSLookup for target ", "target", domain, "error", err)
		result.Err = &tools.ToolError{
			Code:    enums.ToolError,
			Message: fmt.Errorf("error performing DNSLookup %w", err).Error(),
		}
		return result, nil
	}

	result.Result = &lookupResult
	return result, nil
}

func performDNSLookup(ctx context.Context, domain string) (tools.DNSLookupResult, error) {
	var (
		records       []tools.DNSRecord
		DNSSECEnabled bool
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

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return tools.DNSLookupResult{}, ctx.Err()
		default:
			// Proceed with DNS query
		}

		typeRecords, err := QueryDNSRecord(domain, recordType)
		if err != nil {
			return tools.DNSLookupResult{}, err
		}
		records = append(records, typeRecords...)
	}

	// Check if we got a DNSKeyRecord somewhere
	if tools.HasDNSKeyRecord(records) {
		DNSSECEnabled = true
	}

	duration := time.Since(start)

	return tools.DNSLookupResult{
		Domain:         domain,
		DNSRecords:     records,
		DNSSECEnabled:  DNSSECEnabled,
		LookupDuration: duration,
		CreatedAt:      time.Now(),
	}, nil
}

// QueryDNSRecord fetches available records of the specified type and returns TTL information
func QueryDNSRecord(domain string, recordType uint16) ([]tools.DNSRecord, error) {
	var records []tools.DNSRecord

	r := tools.GoogleResolver
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
			records = append(records, tools.DNSRecord{
				Name:  record.Header().Name,
				Type:  tools.ARecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.A.String(),
			})
		case *dns.AAAA:
			records = append(records, tools.DNSRecord{
				Name:  record.Header().Name,
				Type:  tools.AAAARecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.AAAA.String(),
			})
		case *dns.CNAME:
			records = append(records, tools.DNSRecord{
				Name:  record.Header().Name,
				Type:  tools.CNAMERecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Target,
			})
		case *dns.MX:
			records = append(records, tools.DNSRecord{
				Name: record.Hdr.Name,
				Type: tools.MXRecord,
				TTL:  int(record.Hdr.Ttl),
				Value: tools.MailExchange{
					Host:     record.Mx,
					Priority: int(record.Preference),
				},
			})
		case *dns.TXT:
			records = append(records, tools.DNSRecord{
				Name:  record.Hdr.Name,
				Type:  tools.TXTRecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Txt,
			})
		case *dns.NS:
			records = append(records, tools.DNSRecord{
				Name:  record.Hdr.Name,
				Type:  tools.NSRecord,
				TTL:   int(record.Hdr.Ttl),
				Value: record.Ns,
			})
		case *dns.SOA:
			records = append(records, tools.DNSRecord{
				Name: record.Hdr.Name,
				Type: tools.SOARecord,
				TTL:  int(record.Hdr.Ttl),
				Value: tools.StartOfAuthority{
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
			records = append(records, tools.DNSRecord{
				Name: record.Hdr.Name,
				Type: tools.DNSKeyRecord,
				TTL:  int(record.Hdr.Ttl),
				Value: tools.DNSKey{
					Flags:     int(record.Flags),
					Protocol:  int(record.Protocol),
					Algorithm: int(record.Algorithm),
				},
			})
		}
	}
	return records, nil
}

func (s *DNSLookupService) calculateRetryDelay(attempt int) time.Duration {
	// Exponential backoff with jitter
	delay := s.retryDelay * time.Duration(1<<uint(attempt))
	jitter := time.Duration(int64(float64(delay) * 0.2)) // +/- 20% jitter
	if rand.Intn(2) == 0 { // Randomly decide the sign of the jitter
		jitter = -jitter
	}
	delay += jitter

	if delay > 15*time.Second {
		delay = 15 * time.Second
	}
	return delay
}
