package whois

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"golang.org/x/net/publicsuffix"
)

type WhoIsEventResult struct {
	Hosts []whoisparser.WhoisInfo
}

func RunWhoIsScan(targets []string) (*WhoIsEventResult, error) {
	log.Println("Running WhoIs scanner...")

	res := new(WhoIsEventResult)

	for _, target := range targets {
		targetDomain, err := GetDomainFromURL(target)
		if err != nil {
			log.Printf("Error parsing domain from URL for target `%s`: %+v, skipping to the next target.\n", target, err)
			continue
		}

		whoIsRaw, err := GetWhoIsRaw(targetDomain)
		if err != nil {
			log.Printf("Error fetching WHOIS for target `%s`: %+v, skipping to the next target. \n", targetDomain, err)
			continue
		}

		parsedResult, err := whoisparser.Parse(whoIsRaw)
		if err != nil {
			log.Printf("Error parsing WHOIS data for target `%s`: %+v, skipping to the next target. \n", targetDomain, err)
			continue
		}

		// Print the domain status
		if parsedResult.Domain != nil {
			log.Println("Domain:", parsedResult.Domain.Domain)
			log.Println("\tStatus: ", parsedResult.Domain.Status)
			log.Println("\tCreation date: ", parsedResult.Domain.CreatedDate)
			log.Println("\tExpiration date: ", parsedResult.Domain.ExpirationDate)
		}

		if parsedResult.Registrar != nil {
			log.Println(("Registrar:"))
			log.Println("\tName: ", parsedResult.Registrar.Name)

		}

		if parsedResult.Registrant != nil {
			log.Println("Registrant:")
			log.Println("\tName:", parsedResult.Registrant.Name)
			log.Println("\tEmail:", parsedResult.Registrant.Email)
		}
		res.Hosts = append(res.Hosts, parsedResult)
	}

	return res, nil
}

func GetWhoIsRaw(url string) (string, error) {
	whois_raw, err := whois.Whois(url)

	if err != nil {
		return "", err
	}

	return whois_raw, nil
}

func GetDomainFromURL(rawURL string) (string, error) {
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
