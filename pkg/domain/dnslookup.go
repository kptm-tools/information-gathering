// TODO: Maybe create a Record abstract struct... how can we associate a name to a record?
package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type DNSLookupEventResult struct {
	Hosts []DNSLookupResult `json:"hosts"`
}

type DNSLookupResult struct {
	Domain         string            `json:"domain"`          // The domain name being queried
	ARecords       []string          `json:"a_records"`       // IPv4 addresses
	AAAARecords    []string          `json:"aaaa_records"`    // IPv6 addresses
	CNAMERecords   []string          `json:"cname_records"`   // Canonical names
	MXRecords      []MailExchange    `json:"mx_records"`      // Mail exchange records
	TXTRecords     []string          `json:"txt_records"`     // Text records
	NSRecords      []string          `json:"ns_records"`      // Name server records
	SOARecord      *StartOfAuthority `json:"soa_record"`      // Start of authority record
	DNSSECEnabled  bool              `json:"dnssec_enabled"`  // Indicates if DNSSEC is enabled
	TTL            int               `json:"ttl"`             // Time-to-live for the records (in seconds)
	LookupDuration time.Duration     `json:"lookup_duration"` // Time taken to perform the lookup
	CreatedAt      time.Time         `json:"created_at"`      // Timestamp when the lookup was performed
}

// MailExchange represents an MX (Mail Exchange) record.
type MailExchange struct {
	Host     string `json:"host"`     // The mail server host
	Priority int    `json:"priority"` // The priority of the mail server
}

// StartOfAuthority represents an SOA (Start of Authority) record.
type StartOfAuthority struct {
	PrimaryNS  string `json:"primary_ns"`  // Primary name server
	AdminEmail string `json:"admin_email"` // Admin email address
	Serial     int    `json:"serial"`      // Serial number
	Refresh    int    `json:"refresh"`     // Refresh interval (in seconds)
	Retry      int    `json:"retry"`       // Retry interval (in seconds)
	Expire     int    `json:"expire"`      // Expiration limit (in seconds)
	MinimumTTL int    `json:"minimum_ttl"` // Minimum TTL (in seconds)
}

func (r *DNSLookupEventResult) String() string {
	data, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return fmt.Sprintf("Error marshalling DNSLookupEventResult: %v", err)
	}
	return fmt.Sprintf("DNSLookup Event Result\n%s", string(data))
}
