package domain

import (
	"fmt"
	"strings"

	whoisparser "github.com/likexian/whois-parser"
)

type WhoIsEventResult struct {
	Hosts []whoisparser.WhoisInfo `json:"hosts"`
}

func (w *WhoIsEventResult) String() string {
	var sb strings.Builder
	for i, host := range w.Hosts {
		sb.WriteString(fmt.Sprintf("Host %d: \n", i+1))
		if host.Domain != nil {
			sb.WriteString(fmt.Sprintf("  Domain: %s\n", host.Domain.Domain))
			sb.WriteString(fmt.Sprintf("  Status: %v\n", host.Domain.Status))
			sb.WriteString(fmt.Sprintf("  Created: %s\n", host.Domain.CreatedDate))
			sb.WriteString(fmt.Sprintf("  Expires: %s\n", host.Domain.ExpirationDate))
		}
		if host.Registrar != nil {
			sb.WriteString(fmt.Sprintf("  Registrar: %s\n", host.Registrar.Name))
			sb.WriteString(fmt.Sprintf("  Registrar: Email: %s\n", host.Registrar.Email))
			sb.WriteString(fmt.Sprintf("  Registrar: Country: %s\n", host.Registrar.Country))
			sb.WriteString(fmt.Sprintf("  Registrar: Province: %s\n", host.Registrar.Province))
			sb.WriteString(fmt.Sprintf("  Registrar: City: %s\n", host.Registrar.City))
			sb.WriteString(fmt.Sprintf("  Registrar: Street: %s\n", host.Registrar.Street))
			sb.WriteString(fmt.Sprintf("  Registrar: Organization: %s\n", host.Registrar.Organization))
		}
		if host.Registrant != nil {
			sb.WriteString(fmt.Sprintf("  Registrant Name: %s\n", host.Registrant.Name))
			sb.WriteString(fmt.Sprintf("  Registrant Email: %s\n", host.Registrant.Email))
			sb.WriteString(fmt.Sprintf("  Registrant Country: %s\n", host.Registrant.Country))
			sb.WriteString(fmt.Sprintf("  Registrant Province: %s\n", host.Registrant.Province))
			sb.WriteString(fmt.Sprintf("  Registrant City: %s\n", host.Registrant.City))
			sb.WriteString(fmt.Sprintf("  Registrant Street: %s\n", host.Registrant.Street))
			sb.WriteString(fmt.Sprintf("  Registrant Organization: %s\n", host.Registrant.Organization))
		}
		sb.WriteString("\n")

	}
	return sb.String()
}
