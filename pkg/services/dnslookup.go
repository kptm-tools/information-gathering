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
	Logger *slog.Logger
}

var _ interfaces.IDNSLookupService = (*DNSLookupService)(nil)

func NewDNSLookupService() *DNSLookupService {
	return &DNSLookupService{
		Logger: slog.New(slog.Default().Handler()),
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
		s.Logger.Warn("Context canceled during DNS lookup", "domain", domain)
		return tools.ToolResult{}, ctx.Err()
	default:
		// Proceed with the operation
	}

	lookupResult, err := performDNSLookup(ctx, domain)
	if err != nil {
		s.Logger.Error("Error performing DNSLookup for target ", "target", domain, "error", err)
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
