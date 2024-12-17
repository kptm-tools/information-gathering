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
	start := time.Now()

	// Retrieve A and AAAA records
	aRecords, aaaaRecords, err := lookupAandAAAA(domain)
	if err != nil {
		slog.Error(err.Error())
	}

	// Retrieve CNAME records
	cnameRecords, err := lookupCNAME(domain)

	// TXTRecords
	txtRecords, err := lookupTXT(domain)

	// Lookup NSRecords
	nsRecords, err := lookupNS(domain)

	// Retrieve MXRecords
	mxRecords, err := lookupMX(domain)

	// Retrieve SOARecord
	soaRecord, err := lookupSOA(domain)

	// Check is DNSSEC is enabled
	DNSSECEnabled, err := lookupDNSSEC(domain)

	duration := time.Since(start)

	return &dom.DNSLookupResult{
		Domain:         domain,
		ARecords:       aRecords,
		AAAARecords:    aaaaRecords,
		CNAMERecords:   cnameRecords,
		MXRecords:      mxRecords,
		TXTRecords:     txtRecords,
		NSRecords:      nsRecords,
		SOARecord:      soaRecord,
		DNSSECEnabled:  DNSSECEnabled,
		LookupDuration: duration,
		CreatedAt:      time.Now(),
	}, nil

}

func lookupAandAAAA(domain string) ([]string, []string, error) {
	var (
		aRecords    []string
		aaaaRecords []string
	)

	// Retrieve A and AAAA records
	ips, err := net.LookupIP(domain)

	if err != nil {
		return aRecords, aaaaRecords, err
	}

	for _, ip := range ips {
		if ip.To4() != nil {
			aRecords = append(aRecords, ip.String())
		}
		if ip.To4() == nil {
			aaaaRecords = append(aaaaRecords, ip.String())
		}
	}

	return aRecords, aaaaRecords, err
}

func lookupCNAME(domain string) ([]string, error) {
	var cnameRecords []string

	cname, err := net.LookupCNAME(domain)
	if err != nil {
		return cnameRecords, fmt.Errorf("failed to get CNAME record: `%v`", err)
	}
	cnameRecords = append(cnameRecords, cname)
	return cnameRecords, nil
}

func lookupTXT(domain string) ([]string, error) {
	var txtRecords []string

	records, err := net.LookupTXT(domain)
	if err != nil {
		return txtRecords, fmt.Errorf("failed to get TXT records: `%v`", err)
	}
	for _, record := range records {
		txtRecords = append(txtRecords, record)
	}
	return txtRecords, nil

}

func lookupNS(domain string) ([]string, error) {
	var nsRecords []string

	records, err := net.LookupNS(domain)
	if err != nil {
		return nsRecords, fmt.Errorf("failed to get NS records: `%v`\n", err)
	}
	for _, ns := range records {
		nsRecords = append(nsRecords, ns.Host)
	}
	return nsRecords, nil
}

func lookupMX(domain string) ([]dom.MailExchange, error) {
	var mxRecords []dom.MailExchange

	records, err := net.LookupMX(domain)
	if err != nil {
		return mxRecords, fmt.Errorf("failed to get MX records: `%v`\n", err)
	}
	for _, mx := range records {
		r := dom.MailExchange{
			Host:     mx.Host,
			Priority: int(mx.Pref),
		}
		mxRecords = append(mxRecords, r)
	}

	return mxRecords, nil

}

func lookupSOA(domain string) (*dom.StartOfAuthority, error) {
	soaRecord := new(dom.StartOfAuthority)
	dnsServer := "8.8.8.8:53" // GooglePublic DNS server for query

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
		}
	}

	return soaRecord, nil
}

func lookupDNSSEC(domain string) (bool, error) {
	dnsServer := "8.8.8.8:53" // GooglePublic DNS server for query

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeDNSKEY)

	// Use the DNS client to query the server
	client := new(dns.Client)

	response, _, err := client.Exchange(m, dnsServer)
	if err != nil {
		return false, fmt.Errorf("failed to query DNSSECRecords: `%v`", err)
	}

	// Check if DNSKEY records are present
	if len(response.Answer) > 0 {
		// DNSSEC is enabled
		fmt.Printf("DNSSEC is enabled. DNSKEY records for %s:\n", domain)
		for _, answer := range response.Answer {
			if dnskey, ok := answer.(*dns.DNSKEY); ok {
				fmt.Printf("Flags: %d, Protocol: %d, Algorithm: %d\n", dnskey.Flags, dnskey.Protocol, dnskey.Algorithm)
			}
		}
		return true, nil
	}
	return false, nil
}
