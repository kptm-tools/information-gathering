package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// List of User-Agent strings
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Android 10; Mobile; rv:89.0) Gecko/89.0 Firefox/89.0",
}

// Function to get a random User-Agent
func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// HTTP client with random User-Agent
func fetchWithRandomUserAgent(url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the random User-Agent
	rAgent := getRandomUserAgent()
	req.Header.Set("User-Agent", rAgent)

	// Make the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Fetching with user agent failed: %s\n", rAgent)
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	return resp, nil
}

func main() {
	startTime := time.Now()
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
			fmt.Printf("No emails found in page: %s...\n", link[:20])
		} else {
			fmt.Printf("Extracted emails from %s...:\n", link[:20])
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

	fmt.Printf("Execution time: %s\n", time.Since(startTime))

}

func scrapeGoogleLinks(query string) ([]string, error) {
	searchURL := fmt.Sprintf("https://www.google.com/search?num=100&q=%s", url.QueryEscape(query))

	// Make HTTP request
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(searchURL)
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
		// Check attributes
		href, exists := s.Attr("href")
		if exists && strings.Contains(href, "/url?q=") {
			// Clean the Google redirection
			parsedURL, err := url.Parse(href)
			if err != nil {
				fmt.Printf("failed to parse redirection URL %s: %s", href, err.Error())
				return
			}

			cleanLink := parsedURL.Query().Get("q")
			if cleanLink != "" {
				links = append(links, cleanLink)
			}
		}
	})

	return links, nil
}

func extractEmailsFromPage(domain, pageURL string) ([]string, error) {
	doc, err := fetchPageContent(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed fo fetch page: %w", err)
	}

	return extractEmailsFromHTML(domain, doc)

}

// fetchPageContent fetches a page and returns the raw HTML
func fetchPageContent(pageURL string) (*goquery.Document, error) {

	resp, err := fetchWithRandomUserAgent(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to GET page %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-OK HTTP status for %s: %s", pageURL, resp.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}

// extractEmailsFromHTML extracts emails for a given domain from a goquery.Document
func extractEmailsFromHTML(domain string, doc *goquery.Document) ([]string, error) {
	emailSet := make(map[string]struct{})

	emailRegex := buildEmailRegexp(domain)

	// List items
	// Using "*" goes through every HTML element, which is more thorough but less performant
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		// Check text content
		text := decodeHTMLEntities(s.Text())
		emails := emailRegex.FindAllString(text, -1)
		for _, email := range emails {
			// fmt.Println("Found match:", email)
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
	emailRegexpPattern := fmt.Sprintf(`[a-zA-Z0-9._%%+-]+@(?:\w*\.)?%s`, regexp.QuoteMeta(domain))
	return regexp.MustCompile(emailRegexpPattern)
}
