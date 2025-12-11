// Package activities contains all Temporal workflow activities for OSINT discovery
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

// DiscoveryActivities contains all OSINT discovery-related activities
type DiscoveryActivities struct {
	HTTPClient *http.Client
	OSINTUrl   string
	PostgreSQL *sql.DB
}

// DiscoverCompaniesFromGitHub discovers companies from GitHub
func (a *DiscoveryActivities) DiscoverCompaniesFromGitHub(ctx context.Context, query string, maxResults int) ([]CompanyInfo, error) {
	log.Printf("Discovering companies from GitHub: query=%s, max=%d", query, maxResults)

	payload := map[string]interface{}{
		"query":       query,
		"max_results": maxResults,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.OSINTUrl+"/discover/github", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OSINT service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close OSINT response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSINT service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Companies []struct {
			Name        string `json:"name"`
			Domain      string `json:"domain"`
			Description string `json:"description"`
		} `json:"companies"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	companies := make([]CompanyInfo, len(result.Companies))
	for i, c := range result.Companies {
		companies[i] = CompanyInfo{
			Name:        c.Name,
			Domain:      c.Domain,
			Description: c.Description,
			Source:      "github",
		}
		// Store in database
		if err := a.storeCompany(ctx, &companies[i]); err != nil {
			log.Printf("Warning: Failed to store company %s: %v", c.Name, err)
		}
	}

	log.Printf("Discovered %d companies from GitHub", len(companies))
	return companies, nil
}

// storeCompany stores a discovered company in PostgreSQL
func (a *DiscoveryActivities) storeCompany(ctx context.Context, company *CompanyInfo) error {
	if a.PostgreSQL == nil {
		log.Println("Warning: PostgreSQL connection not available, skipping storage")
		return nil
	}

	var companyID int
	err := a.PostgreSQL.QueryRowContext(ctx, `
		INSERT INTO companies (name, domain, description, source)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (domain) DO UPDATE 
		SET name = EXCLUDED.name,
		    description = COALESCE(EXCLUDED.description, companies.description)
		RETURNING id
	`, company.Name, company.Domain, company.Description, company.Source).Scan(&companyID)

	if err != nil {
		return fmt.Errorf("failed to store company: %w", err)
	}

	log.Printf("✅ Stored company: %s (ID: %d)", company.Name, companyID)
	return nil
}

// DiscoverCompaniesFromGoogleDorks discovers companies using Google dorks
func (a *DiscoveryActivities) DiscoverCompaniesFromGoogleDorks(ctx context.Context, query string, maxResults int) ([]CompanyInfo, error) {
	log.Printf("Discovering companies from Google Dorks: query=%s, max=%d", query, maxResults)

	payload := map[string]interface{}{
		"query":       query,
		"max_results": maxResults,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.OSINTUrl+"/discover/google-dork", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OSINT service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close OSINT response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSINT service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		URLs []struct {
			URL    string `json:"url"`
			Domain string `json:"domain"`
		} `json:"urls"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Group by domain to get unique companies
	domainMap := make(map[string]CompanyInfo)
	for _, u := range result.URLs {
		if _, exists := domainMap[u.Domain]; !exists {
			domainMap[u.Domain] = CompanyInfo{
				Name:   u.Domain,
				Domain: u.Domain,
				Source: "google_dork",
			}
		}
	}

	companies := make([]CompanyInfo, 0, len(domainMap))
	for _, c := range domainMap {
		companies = append(companies, c)
	}

	log.Printf("Discovered %d companies from Google Dorks", len(companies))
	return companies, nil
}

// AddCompanyManually adds a single company manually
func (a *DiscoveryActivities) AddCompanyManually(ctx context.Context, domain string) ([]CompanyInfo, error) {
	log.Printf("Adding company manually: %s", domain)

	// Simply return the domain as a company
	return []CompanyInfo{
		{
			Name:   domain,
			Domain: domain,
			Source: "manual",
		},
	}, nil
}

// DiscoverCareerPages discovers career pages for a company domain
func (a *DiscoveryActivities) DiscoverCareerPages(ctx context.Context, domain string, companyName string) ([]CareerPageInfo, error) {
	log.Printf("Discovering career pages for: %s (%s)", domain, companyName)

	payload := map[string]string{
		"domain": domain,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.OSINTUrl+"/discover/career-pages", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OSINT service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close OSINT response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSINT service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Pages []struct {
			URL        string  `json:"url"`
			PageType   string  `json:"page_type"`
			Confidence float64 `json:"confidence"`
		} `json:"career_pages"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	pages := make([]CareerPageInfo, len(result.Pages))
	for i, p := range result.Pages {
		pages[i] = CareerPageInfo{
			URL:        p.URL,
			Domain:     domain,
			PageType:   p.PageType,
			Confidence: p.Confidence,
			Priority:   1,
		}
	}

	log.Printf("Discovered %d career pages for %s", len(pages), domain)
	return pages, nil
}

// EnumerateSubdomains enumerates subdomains for a domain
func (a *DiscoveryActivities) EnumerateSubdomains(ctx context.Context, domain string) ([]CareerPageInfo, error) {
	log.Printf("Enumerating subdomains for: %s", domain)

	payload := map[string]string{
		"domain": domain,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.OSINTUrl+"/discover/subdomains", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OSINT service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close OSINT response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSINT service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Subdomains []string `json:"subdomains"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert subdomains to potential career page URLs
	pages := make([]CareerPageInfo, len(result.Subdomains))
	for i, subdomain := range result.Subdomains {
		pages[i] = CareerPageInfo{
			URL:        "https://" + subdomain,
			Domain:     domain,
			PageType:   "subdomain",
			Confidence: 0.5,
			Priority:   2,
		}
	}

	log.Printf("Found %d subdomains for %s", len(pages), domain)
	return pages, nil
}

// DetectATS detects ATS platform for a URL
func (a *DiscoveryActivities) DetectATS(ctx context.Context, url string) (ATSInfo, error) {
	log.Printf("Detecting ATS for: %s", url)

	payload := map[string]string{
		"url": url,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return ATSInfo{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.OSINTUrl+"/detect/ats", bytes.NewBuffer(body))
	if err != nil {
		return ATSInfo{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return ATSInfo{}, fmt.Errorf("failed to call OSINT service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close OSINT response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return ATSInfo{}, fmt.Errorf("OSINT service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		IsATS      bool    `json:"is_ats"`
		Platform   string  `json:"platform"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ATSInfo{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return ATSInfo{
		URL:        url,
		IsATS:      result.IsATS,
		Platform:   result.Platform,
		Confidence: result.Confidence,
	}, nil
}

// QueueURLsForCrawling queues discovered URLs for the crawler
func (a *DiscoveryActivities) QueueURLsForCrawling(ctx context.Context, pages []CareerPageInfo) (int, error) {
	log.Printf("Queuing %d URLs for crawling", len(pages))

	if a.PostgreSQL == nil {
		log.Println("Warning: PostgreSQL connection not available, skipping storage")
		return len(pages), nil
	}

	queued := 0
	for _, page := range pages {
		// Generate URL hash
		hash := sha256.Sum256([]byte(page.URL))
		urlHash := hex.EncodeToString(hash[:])

		// Get company ID from domain
		var companyID *int
		err := a.PostgreSQL.QueryRowContext(ctx, `
			SELECT id FROM companies WHERE domain = $1 LIMIT 1
		`, page.Domain).Scan(&companyID)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error finding company for domain %s: %v", page.Domain, err)
			continue
		}

		// Insert discovered URL
		_, err = a.PostgreSQL.ExecContext(ctx, `
			INSERT INTO discovered_urls (
				company_id, url, url_hash, page_type, confidence,
				ats_platform, discovered_via, priority
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (url_hash) DO UPDATE
			SET confidence = GREATEST(discovered_urls.confidence, EXCLUDED.confidence),
			    priority = GREATEST(discovered_urls.priority, EXCLUDED.priority)
		`, companyID, page.URL, urlHash, page.PageType, page.Confidence,
			page.ATSPlatform, "osint", page.Priority)

		if err != nil {
			log.Printf("Warning: Failed to queue URL %s: %v", page.URL, err)
			continue
		}

		queued++
	}

	log.Printf("✅ Queued %d/%d URLs for crawling", queued, len(pages))
	return queued, nil
}

// GenerateDorkQueries generates Google dork queries for a keyword
func (a *DiscoveryActivities) GenerateDorkQueries(_ context.Context, keyword string) ([]string, error) {
	log.Printf("Generating dork queries for keyword: %s", keyword)

	// Generate common dork patterns
	queries := []string{
		fmt.Sprintf("intext:\"%s\" AND (\"careers\" OR \"jobs\")", keyword),
		fmt.Sprintf("site:greenhouse.io \"%s\"", keyword),
		fmt.Sprintf("site:lever.co \"%s\"", keyword),
		fmt.Sprintf("site:ashbyhq.com \"%s\"", keyword),
		fmt.Sprintf("inurl:careers \"%s\"", keyword),
		fmt.Sprintf("\"we are hiring\" \"%s\"", keyword),
	}

	return queries, nil
}

// ExecuteDorkQuery executes a Google dork query using SERPAPI
func (a *DiscoveryActivities) ExecuteDorkQuery(ctx context.Context, query string, maxResults int) ([]string, error) {
	log.Printf("Executing dork query: %s", query)

	payload := map[string]interface{}{
		"query":       query,
		"max_results": maxResults,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.OSINTUrl+"/discover/google-dork", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OSINT service: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close OSINT response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OSINT service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		URLs []struct {
			URL string `json:"url"`
		} `json:"urls"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	urls := make([]string, len(result.URLs))
	for i, u := range result.URLs {
		urls[i] = u.URL
	}

	return urls, nil
}

// DetectATSAndExtractDomain detects ATS and extracts domain from URL
func (a *DiscoveryActivities) DetectATSAndExtractDomain(ctx context.Context, url string) (CareerPageInfo, error) {
	atsInfo, err := a.DetectATS(ctx, url)
	if err != nil {
		return CareerPageInfo{}, err
	}

	return CareerPageInfo{
		URL:         url,
		ATSPlatform: atsInfo.Platform,
		Confidence:  atsInfo.Confidence,
		Priority:    1,
	}, nil
}

// Data structures for discovery activities
type CompanyInfo struct {
	Name        string
	Domain      string
	Description string
	Source      string
}

type CareerPageInfo struct {
	URL         string
	Domain      string
	PageType    string
	Confidence  float64
	ATSPlatform string
	Priority    int
}

type ATSInfo struct {
	URL        string
	IsATS      bool
	Platform   string
	Confidence float64
}
