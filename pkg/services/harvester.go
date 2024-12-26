package services

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	cmmn "github.com/kptm-tools/common/common/results"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
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

// Job defines a job struct with attributes accessed by workers to extract emails
type Job struct {
	Link   string // The Link or URL to scrape
	Domain string // The name of the domain being scanned
}

// Result defines the Result of a HarvestEmails Job
type Result struct {
	Emails []string
	Error  error
}

func worker(jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		emails, err := extractEmailsFromPage(job.Domain, job.Link)
		results <- Result{Emails: emails, Error: err}
	}
}

type HarvesterService struct {
	Logger *slog.Logger
}

var _ interfaces.IHarvesterService = (*HarvesterService)(nil)

func NewHarvesterService() *HarvesterService {
	return &HarvesterService{
		Logger: slog.New(slog.Default().Handler()),
	}
}

func (s *HarvesterService) RunScan(targets []string) (*[]cmmn.TargetResult, error) {
	var (
		tResults []cmmn.TargetResult
		errs     []error
	)

	// To avoid rate-limiting, we don't use coroutines here
	for _, target := range targets {

		emails, err := s.HarvestEmails(target)
		if err != nil {
			s.Logger.Error("Error harvesting emails ", target, err)
			errs = append(errs, err)
			continue
		}

		tRes := cmmn.TargetResult{
			Target:  target,
			Results: map[string]interface{}{"harvester": emails},
		}

		tResults = append(tResults, tRes)
	}

	// fmt.Printf("Got emails: %v", emails)

	return &tResults, nil
}

func (s *HarvesterService) HarvestEmails(target string) ([]string, error) {
	startTime := time.Now()

	var links []string

	linkedInLinks, err := scrapeLinkedinLinks(target)
	if err != nil {
		return nil, fmt.Errorf("error scraping linkedin links: %v", err)
	}

	time.Sleep(5 * time.Second)
	googleLinks, err := scrapeGoogleLinks(target)
	if err != nil {
		return nil, fmt.Errorf("error scraping google links: %v", err)
	}

	links = append(links, linkedInLinks...)
	links = append(links, googleLinks...)
	log.Printf("Found %d total links...\n", len(links))

	// Create channels for jobs and results
	numWorkers := 5
	jobs := make(chan Job, len(links))
	results := make(chan Result, len(links))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(jobs, results, &wg)
	}

	// Send jobs to the jobs channel
	go func() {
		for _, link := range links {
			jobs <- Job{Link: link, Domain: target}
		}
		close(jobs)
	}()

	// Wait for all workers to finish and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allEmails []string
	for result := range results {
		if result.Error != nil {
			// fmt.Printf("Error processing link: %v", result.Error)
			continue
		}
		allEmails = append(allEmails, result.Emails...)
	}

	uniqueEmails := removeDuplicateEmails(allEmails)
	fmt.Println("The following emails have been extracted:")
	for _, email := range uniqueEmails {
		fmt.Printf("\t%s\n", email)
	}
	fmt.Println("Total unique emails extracted:", len(uniqueEmails))

	fmt.Printf("Execution time: %s\n", time.Since(startTime))

	return uniqueEmails, nil
}

func (s *HarvesterService) HarvestSubdomains(target string) ([]string, error) {
	// TODO: Implementation pending
	return nil, nil
}

// Create a reusable HTTPClient
func createHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// Create a reusable HTTP request with custom headers
func createHTTPRequest(method, url string, headers map[string]string) (*http.Request, error) {

	if method == "" || !isValidHTTPMethod(method) {
		return nil, fmt.Errorf("invalid HTTP method")
	}

	if url == "" {
		return nil, fmt.Errorf("url cannot empty")
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return req, nil

}

// Helper function to validate HTTP methods
func isValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// Perform a GET request with custom HTTP headers
func fetchWithCustomHeaders(url string, headers map[string]string) (*http.Response, error) {
	client := createHTTPClient(5 * time.Second)
	req, err := createHTTPRequest("GET", url, headers)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	return resp, nil
}

// Function to get a random User-Agent
func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// HTTP client with random User-Agent
func fetchWithRandomUserAgent(url string) (*http.Response, error) {
	headers := map[string]string{
		"User-Agent": getRandomUserAgent(),
	}
	return fetchWithCustomHeaders(url, headers)
}

func scrapeGoogleLinks(query string) ([]string, error) {
	searchURL := fmt.Sprintf("https://www.google.com/search?num=100&q=%s", url.QueryEscape(query))

	fmt.Println("Scraping Google Links...")
	// Make HTTP request
	resp, err := fetchWithRandomUserAgent(searchURL)
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

	fmt.Printf("Found %d Google links...\n", len(links))
	return links, nil
}

func scrapeLinkedinLinks(query string) ([]string, error) {
	searchURL := fmt.Sprintf(
		"https://www.google.com/search?num=100&q=site:linkedin.com+%s",
		url.QueryEscape(query),
	)

	fmt.Println("Scraping LinkedIn links with Google...")
	resp, err := fetchWithRandomUserAgent(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch linkedin search results: %w", err)
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
	doc.Find("a").Each(func(i int, sel *goquery.Selection) {
		// Check attributes
		href, exists := sel.Attr("href")
		fmt.Printf("Found href: %s attrib on a attr: %s", href, sel.Text())
		if exists && !strings.HasPrefix(href, "/search?q=site:") {
			// Clean the Google redirection
			parsedURL, err := url.Parse(href)
			if err != nil {
				fmt.Printf("failed to parse redirection URL %s: %s", href, err)
				return
			}

			urlString := parsedURL.String()
			if !strings.HasPrefix(urlString, "https://") {
				fmt.Printf("only HTTPS connections are allowed: %s", urlString)
				return
			}

			links = append(links, urlString)
		}
	})

	fmt.Printf("Found %d LinkedIn links...\n", len(links))
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