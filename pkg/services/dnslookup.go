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

	var (
		records       []dom.DNSRecord
		DNSSECEnabled bool
	)
	start := time.Now()

	// Retrieve A and AAAA records
	aRecords, err := lookupAandAAAA(domain)
	if err == nil {
		records = append(records, aRecords...)
	}

	// Retrieve CNAME records
	cnameRecords, err := lookupCNAME(domain)
	if err == nil {
		records = append(records, cnameRecords...)
	}

	// TXTRecords
	txtRecords, err := lookupTXT(domain)
	if err == nil {
		records = append(records, txtRecords...)
	}

	// Lookup NSRecords
	nsRecords, err := lookupNS(domain)
	if err == nil {
		records = append(records, nsRecords...)
	}

	// Retrieve MXRecords
	mxRecords, _ := lookupMX(domain)
	if err == nil {
		records = append(records, mxRecords...)
	}

	// Retrieve SOARecord
	soaRecords, err := lookupSOA(domain)
	if err == nil {
		records = append(records, soaRecords...)
	}

	// Retrieve DNSSECRecord
	DNSSECRecords, _ := lookupDNSSEC(domain)
	// Check is DNSSEC is enabled
	if len(DNSSECRecords) > 0 && err == nil {
		records = append(records, DNSSECRecords...)
		DNSSECEnabled = true
	}

	duration := time.Since(start)

	return &dom.DNSLookupResult{
		Domain:         domain,
		DNSRecords:     records,
		DNSSECEnabled:  DNSSECEnabled,
		LookupDuration: duration,
		CreatedAt:      time.Now(),
	}, nil

}

func lookupAandAAAA(domain string) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord

	// Retrieve A and AAAA records
	ips, err := net.LookupIP(domain)
	if err != nil {
		return records, err
	}

	for _, ip := range ips {
		recordType := dom.ARecord
		if ip.To4() == nil {
			recordType = dom.AAAARecord
			records = append(records, dom.DNSRecord{
				Type:  recordType,
				Name:  domain,
				Value: ip.String(),
			})
		}
	}
	return records, err
}

func lookupCNAME(domain string) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord

	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return records, fmt.Errorf("failed to get CNAME record: `%v`", err)
	}
	records = append(records, dom.DNSRecord{
		Type:  dom.CNAMERecord,
		Name:  domain,
		Value: cname,
	})
	return records, nil
}

func lookupTXT(domain string) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord

	txtRecords, err := net.LookupTXT(domain)
	if err != nil {
		return records, fmt.Errorf("failed to get TXT records: `%v`", err)
	}
	for _, txt := range txtRecords {
		records = append(records, dom.DNSRecord{
			Type:  dom.TXTRecord,
			Name:  domain,
			Value: txt,
		})
	}
	return records, nil
}

func lookupNS(domain string) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord

	nsRecords, err := net.LookupNS(domain)
	if err != nil {
		return records, fmt.Errorf("failed to get NS records: `%v`\n", err)
	}
	for _, ns := range nsRecords {
		records = append(records, dom.DNSRecord{
			Type:  dom.NSRecord,
			Name:  domain,
			Value: ns.Host,
		})
	}
	return records, nil
}

func lookupMX(domain string) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord

	mxRecords, err := net.LookupMX(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to get MX records: `%v`\n", err)
	}
	for _, mxRecord := range mxRecords {
		mx := dom.MailExchange{
			Host: mxRecord.Host,
		}

		priorityInt := int(mxRecord.Pref)
		record := dom.DNSRecord{
			Type:  dom.MXRecord,
			Name:  domain,
			Value: mx,
		}
		record.Priority = &priorityInt

		records = append(records, record)
	}

	return records, nil

}

func lookupSOA(domain string) ([]dom.DNSRecord, error) {
	soaRecord := new(dom.StartOfAuthority)
	dnsServer := "8.8.8.8:53" // GooglePublic DNS server for query
	var ttl int

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeSOA)

	// Use the DNS client to query the server
	client := new(dns.Client)
	response, _, err := client.Exchange(m, dnsServer)
	if err != nil {
		return nil, fmt.Errorf("Error querying DNSServer: `%v`", err)
	}

	for _, answer := range response.Answer {
		if soa, ok := answer.(*dns.SOA); ok {
			soaRecord.PrimaryNS = soa.Ns
			soaRecord.AdminEmail = soa.Mbox
			soaRecord.Serial = int(soa.Serial)
			soaRecord.Refresh = int(soa.Refresh)
			soaRecord.Retry = int(soa.Retry)
			soaRecord.Expire = int(soa.Expire)
			soaRecord.MinimumTTL = int(soa.Minttl)
			ttl = int(soa.Hdr.Ttl)
		}
	}

	return []dom.DNSRecord{{
		Type:  dom.SOARecord,
		Name:  domain,
		Value: soaRecord,
		TTL:   ttl,
	}}, nil
}

func lookupDNSSEC(domain string) ([]dom.DNSRecord, error) {
	var records []dom.DNSRecord
	dnsKey := new(dom.DNSKey)
	dnsServer := "8.8.8.8:53" // GooglePublic DNS server for query

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeDNSKEY)

	// Use the DNS client to query the server
	client := new(dns.Client)

	response, _, err := client.Exchange(m, dnsServer)
	if err != nil {
		return nil, fmt.Errorf("failed to query DNSSECRecords: `%v`", err)
	}

	// Check if DNSKEY records are present
	if len(response.Answer) > 0 {
		// DNSSEC is enabled
		fmt.Printf("DNSSEC is enabled. DNSKEY records for %s:\n", domain)
		for _, answer := range response.Answer {
			if dnskey, ok := answer.(*dns.DNSKEY); ok {
				fmt.Printf("Flags: %d, Protocol: %d, Algorithm: %d\n", dnskey.Flags, dnskey.Protocol, dnskey.Algorithm)
				dnsKey.Protocol = int(dnskey.Protocol)
				dnsKey.Flags = int(dnskey.Flags)
				dnsKey.Algorithm = int(dnskey.Algorithm)
			}
			records = append(records, dom.DNSRecord{
				Type:  dom.DNSSECRecord,
				Name:  domain,
				Value: dnsKey,
				TTL:   int(answer.Header().Ttl),
			})
		}
		return records, nil
	}
	return nil, nil
}
