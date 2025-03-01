package services

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kptm-tools/common/common/pkg/enums"
	"github.com/kptm-tools/common/common/pkg/results/tools"
	"github.com/kptm-tools/information-gathering/pkg/interfaces"
	"golang.org/x/net/html"
)

// List of User-Agent strings
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.4; rv:124.0) Gecko/20100101 Firefox/124.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux i686; rv:124.0) Gecko/20100101 Firefox/124.0",
}

var wordLists = map[wordListSize]string{
	WordListMedium: "subdomains-1000.txt",
	WordListLarge:  "subdomains-10000.txt",
}

const (
	WordListLarge  wordListSize = "large"
	WordListMedium wordListSize = "medium"
)

type wordListSize string

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

type HarvesterService struct {
	Logger     *slog.Logger
	HTTPClient *http.Client
}

var _ interfaces.IHarvesterService = (*HarvesterService)(nil)

func NewHarvesterService() *HarvesterService {
	return &HarvesterService{
		Logger:     slog.New(slog.Default().Handler()),
		HTTPClient: createHTTPClient(10 * time.Second),
	}
}

func (s *HarvesterService) RunScan(ctx context.Context, target string) (tools.ToolResult, error) {

	// To avoid rate-limiting, we don't use coroutines here
	result := tools.ToolResult{
		Tool:      enums.ToolHarvester,
		Result:    &tools.HarvesterResult{},
		Timestamp: time.Now().UTC(),
	}

	select {
	case <-ctx.Done():
		s.Logger.Warn("Context canceled during Harvester search", "target", target)
		return tools.ToolResult{}, ctx.Err()
	default:
		// Proceed with operation
	}

	emails, err := s.HarvestEmails(ctx, target)
	if err != nil {
		s.Logger.Error("Error harvesting emails", "target", target, "error", err)
		result.Err = &tools.ToolError{
			Code:    enums.ToolError,
			Message: fmt.Sprintf("error harvesting emails: %s", err.Error()),
		}

		return result, nil
	}
	result.Result = &tools.HarvesterResult{
		Emails: emails,
	}

	subdomains, err := s.HarvestSubdomains(ctx, target)
	if err != nil {
		s.Logger.Error("Error harvesting subdomains", "target", target, "error", err)
		result.Err = &tools.ToolError{
			Code:    enums.ToolError,
			Message: fmt.Sprintf("error harvesting subdomains: %s", err.Error()),
		}

		return result, nil
	}

	result.Result = &tools.HarvesterResult{
		Emails:     emails,
		Subdomains: subdomains,
	}

	return result, nil
}

// HarvestEmails extracts emails from a target
func (s *HarvesterService) HarvestEmails(ctx context.Context, domain string) ([]string, error) {
	startTime := time.Now()
	s.Logger.Info("Harvesting emails", "target", domain)

	links, err := s.scrapeLinks(ctx, domain)
	if err != nil {
		s.Logger.Error("Failed to scrape links", "target", domain, "error", err)
		return nil, err
	}

	allEmails := s.processLinks(links, domain, ctx)
	uniqueEmails := removeDuplicateEmails(allEmails)

	s.Logger.Info("Completed email harvesting",
		"target", domain,
		"email_count", len(uniqueEmails),
		"unique_emails", uniqueEmails,
		"duration", time.Since(startTime).String(),
	)

	return uniqueEmails, nil
}

func (s *HarvesterService) HarvestSubdomains(ctx context.Context, domain string) ([]string, error) {
	startTime := time.Now()
	s.Logger.Info("Harvesting subdomains", "target", domain)

	words, err := s.readWordList(WordListMedium)
	if err != nil {
		return nil, err
	}

	var (
		subdomains []string
		wg         sync.WaitGroup
		results    = make(chan string, len(words))
		sem        = make(chan struct{}, 5) // Semaphore for rate limiting (5 goroutines at a time)
	)

	// Start a goroutine for collecting results
	done := make(chan struct{})
	go func() {
		defer close(done)
		for subdomain := range results {
			subdomains = append(subdomains, subdomain)
		}
	}()

	// Process each word
wordLoop:
	for _, word := range words {
		select {
		case <-ctx.Done():
			s.Logger.Warn("Subdomain harvest cancelled")
			break wordLoop
		default:
			// Proceed with operation
		}

		wg.Add(1)
		go func(word string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			// Check for cancellation
			select {
			case <-ctx.Done():
				s.Logger.Warn("Context cancelled in subdomain goroutine", "word", word)
				return
			default:
				// Proceed with subdomain processing
			}

			subdomain, err := s.processSubdomain(ctx, word, domain)
			if err != nil {
				// Subdomain does not exist
				// s.Logger.Debug("Subdomain does not exist", "word", word, "error", err)
				return
			}

			// Send result to channel if not cancelled
			select {
			case results <- subdomain:
			case <-ctx.Done():
				s.Logger.Warn("Context canceled before writing result", "word", word)
			}
		}(word)
	}

	// Close results channel once all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	<-done

	s.Logger.Info("Completed subdomain harvesting",
		"target", domain,
		"subdomain_count", len(subdomains),
		"subdomains", subdomains,
		"duration", time.Since(startTime).String(),
	)
	return subdomains, nil
}

// readWordList reads a wordList file by its size (e.g., Large, medium, small)
func (s *HarvesterService) readWordList(size wordListSize) ([]string, error) {
	fileName, exists := wordLists[size]
	if !exists {
		return nil, fmt.Errorf("wordlist not found for key: %s", size)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(wd, "pkg", "services", "wordlists", fileName)

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var subdomains []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		subdomains = append(subdomains, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return subdomains, nil
}

// processSubdomain processes a single word to check if the subdomain exists
func (s *HarvesterService) processSubdomain(ctx context.Context, word, domain string) (string, error) {
	url := fmt.Sprintf("http://%s.%s", word, domain)

	req, err := createHTTPRequest("GET", url, map[string]string{})
	if err != nil {
		return "", err
	}

	req = req.WithContext(ctx)

	res, err := s.HTTPClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("subdomain check timed out: %w", ctx.Err())
		}
		return "", fmt.Errorf("failed to check subdomain %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		s.Logger.Info("✅ Found subdomain", "subdomain", url)
		return url, nil
	}

	return "", nil

}

func (s *HarvesterService) scrapeLinks(ctx context.Context, target string) ([]string, error) {
	s.Logger.Debug("Scraping Google and LinkedIn links", "target", target)

	googleLinks, err := scrapeGoogleLinks(ctx, target, s.HTTPClient)
	if err != nil {
		s.Logger.Warn("Standard Google scraping failed", "target", target, "error", err)
	}

	randomTimeout(2, 5)
	linkedInLinks, err := scrapeLinkedinLinks(ctx, target, s.HTTPClient)
	if err != nil {
		s.Logger.Warn("Google scraping for LinkedIn links failed", "target", target, "error", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	links := append(googleLinks, linkedInLinks...)
	s.Logger.Debug("Scraped links", "target", target, "link_count", len(links))

	return links, nil
}

func (s *HarvesterService) processLinks(links []string, domain string, ctx context.Context) []string {

	// Create channels for jobs and results
	numWorkers := 5
	jobs := make(chan Job, len(links))
	results := make(chan Result, len(links))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go s.worker(jobs, results, &wg, ctx)
	}

	// Send jobs to the jobs channel
	go func() {
		for _, link := range links {
			jobs <- Job{Link: link, Domain: domain}
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
			s.Logger.Debug("Error processing link", "link", result.Error)
			continue
		}
		allEmails = append(allEmails, result.Emails...)
	}
	return allEmails
}

func (s *HarvesterService) worker(jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			results <- Result{Emails: nil, Error: ctx.Err()}
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			emails, err := extractEmailsFromPage(ctx, job.Domain, job.Link, s.HTTPClient)
			results <- Result{Emails: emails, Error: err}
		}
	}
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
func fetchWithCustomHeaders(ctx context.Context, url string, headers map[string]string, client *http.Client) (*http.Response, error) {
	req, err := createHTTPRequest("GET", url, headers)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

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
func fetchWithRandomUserAgent(ctx context.Context, url string, client *http.Client) (*http.Response, error) {
	headers := map[string]string{
		"User-Agent": getRandomUserAgent(),
	}
	return fetchWithCustomHeaders(ctx, url, headers, client)
}

func scrapeGoogleLinks(ctx context.Context, query string, client *http.Client) ([]string, error) {
	searchURL := fmt.Sprintf("https://www.google.com/search?num=100&q=%s", url.QueryEscape(query))
	slog.Debug("Searching Google links...", slog.String("search_url", searchURL))

	// Make HTTP request
	resp, err := fetchWithRandomUserAgent(ctx, searchURL, client)
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
		// fmt.Printf("Found href: %s attrib on a attr: %s\n", href, s.Text())
		if exists && !strings.HasPrefix(href, "/search?q=") {
			// Clean the Google redirection
			parsedURL, err := url.Parse(href)
			if err != nil {
				fmt.Printf("failed to parse redirection URL %s: %s", href, err.Error())
				return
			}

			urlString := parsedURL.String()
			if !strings.HasPrefix(urlString, "https://") {
				// fmt.Printf("only HTTPS connections are allowed: %s\n", urlString)
				return
			}
			links = append(links, urlString)
		}
	})

	slog.Debug("Found google links",
		slog.Int("link_count", len(links)),
		slog.Any("links", links))

	return links, nil
}

func scrapeLinkedinLinks(ctx context.Context, query string, client *http.Client) ([]string, error) {
	searchURL := fmt.Sprintf(
		"https://www.google.com/search?num=100&q=site:linkedin.com+%s",
		url.QueryEscape(query),
	)

	resp, err := fetchWithRandomUserAgent(ctx, searchURL, client)
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
		if exists && !strings.HasPrefix(href, "/search?q=site:") {
			// Clean the Google redirection
			parsedURL, err := url.Parse(href)
			if err != nil {
				return
			}

			urlString := parsedURL.String()
			if !strings.HasPrefix(urlString, "https://") {
				return
			}

			links = append(links, urlString)
		}
	})

	return links, nil
}

func extractEmailsFromPage(ctx context.Context, domain, pageURL string, client *http.Client) ([]string, error) {
	doc, err := fetchPageContent(ctx, pageURL, client)
	if err != nil {
		return nil, fmt.Errorf("failed fo fetch page: %w", err)
	}

	return extractEmailsFromHTML(domain, doc)

}

// fetchPageContent fetches a page and returns the raw HTML
func fetchPageContent(ctx context.Context, pageURL string, client *http.Client) (*goquery.Document, error) {

	resp, err := fetchWithRandomUserAgent(ctx, pageURL, client)
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

// randomTimeout does time.Sleep() for a random amount of seconds between min and max
func randomTimeout(min, max int) {
	d := rand.Intn(max-min) + min
	time.Sleep(time.Second * time.Duration(d))
}
