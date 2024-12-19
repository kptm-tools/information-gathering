package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func main() {
	fmt.Println("Hello Web Scraper!")

	targetDomain := "aynitech.com"
	fmt.Println("Scraping Google Links...")
	links, err := scrapeGoogleLinks(targetDomain)
	if err != nil {
		log.Fatalf("Error scraping Google links: %v", err)
	}
	fmt.Printf("Found %d links...\n", len(links))

	var emails []string
	for _, link := range links {
		foundEmails, err := extractEmailsFromPage(targetDomain, link)
		if err != nil {
			log.Printf("Error extracting emails from page %s: %v", link, err)
			continue
		}

		if len(foundEmails) == 0 {
			fmt.Println("No emails found")
		} else {
			fmt.Println("Extracted emails:")
			for _, email := range foundEmails {
				fmt.Printf("\t%s\n", email)
			}
		}

		emails = append(emails, foundEmails...)
	}
	fmt.Println("The following emails have been extracted:")
	uniqueEmails := removeDuplicateEmails(emails)
	for _, email := range uniqueEmails {
		fmt.Printf("\t%s\n", email)
	}

}

func scrapeGoogleLinks(query string) ([]string, error) {
	searchURL := fmt.Sprintf("https://www.google.com/search?num=100&q=%s", url.QueryEscape(query))

	// Make HTTP request
	resp, err := http.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var links []string
	// List items
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		// Check text content
		// Check attributes
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, "/url?q=") {
			// Clean the Google redirection
			cleanLink := strings.Split(strings.TrimPrefix(href, "/url?q="), "&")
			if len(cleanLink) > 0 {
				links = append(links, cleanLink[0])
			}
		}
	})

	return links, nil
}

func extractEmailsFromPage(domain, pageURL string) ([]string, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK HTTP status for %s: %s", pageURL, resp.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	emailSet := make(map[string]struct{})

	emailRegex := buildEmailRegexp(domain)

	// List items
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Check text content
		text := decodeHTMLEntities(s.Text())
		emails := emailRegex.FindAllString(text, -1)
		for _, email := range emails {
			fmt.Println("Found match:", email)
			emailSet[email] = struct{}{}
		}

		for _, attr := range []string{"href", "src", "data-email", "content", "value", "alt", "placeholder"} {
			// Check attributes
			value, exists := s.Attr(attr)
			if exists {
				// Match emails with regex
				matches := emailRegex.FindAllString(value, -1)
				for _, email := range matches {
					emailSet[email] = struct{}{}
				}
			}
		}
	})

	// Convert set to slice
	uniqueEmails := make([]string, 0, len(emailSet))
	for email := range emailSet {
		uniqueEmails = append(uniqueEmails, email)
	}

	return uniqueEmails, nil
}

func removeDuplicateEmails(slice []string) []string {
	seen := make(map[string]bool)
	res := []string{}

	for _, val := range slice {
		if _, ok := seen[val]; !ok {
			seen[val] = true
			res = append(res, val)
		}
	}
	return res
}

// decodeHTMLEntities decodes HTML entities such as '&#64' to '@'
func decodeHTMLEntities(input string) string {
	return html.UnescapeString(input)
}

// buildEmailRegexp builds a regular expression to extract emails associated to a domain
func buildEmailRegexp(domain string) *regexp.Regexp {
	emailRegexpPattern := fmt.Sprintf(`[a-zA-Z0-9._%%+-]+@%s`, regexp.QuoteMeta(domain))
	return regexp.MustCompile(emailRegexpPattern)
}
