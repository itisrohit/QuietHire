package parsers

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// NotionParser handles job parsing from Notion pages
type NotionParser struct{}

// CanParse checks if URL is a Notion job page
func (p *NotionParser) CanParse(url string) bool {
	return strings.Contains(url, "notion.site") ||
		strings.Contains(url, "notion.so")
}

// Parse extracts job data from Notion page
func (p *NotionParser) Parse(page playwright.Page, url string) (*JobData, error) {
	job := &JobData{
		URL:    url,
		Source: "notion",
	}

	// Extract job title (Notion uses h1 for page titles)
	title, err := ExtractText(page, "h1.notranslate, h1, [data-content-editable-leaf='true']")
	if err == nil {
		job.Title = CleanText(title)
	}

	// Extract company name from metadata or page content
	company, err := ExtractText(page, "meta[property='og:site_name'], [class*='company'], strong:contains('Company')")
	if err == nil {
		job.Company = CleanText(company)
	}

	// Extract location
	location, err := ExtractText(page, "[class*='location'], strong:contains('Location')")
	if err == nil {
		job.Location = CleanText(location)

		// Check if remote
		locationLower := strings.ToLower(location)
		if strings.Contains(locationLower, "remote") {
			job.Remote = true
		}
	}

	// Extract job description (Notion pages have content in various blocks)
	desc, err := ExtractText(page, "article, [class*='notion-page-content'], div[data-block-id]")
	if err == nil {
		job.Description = CleanText(desc)
	}

	// Extract salary range
	salary, err := ExtractText(page, "strong:contains('Salary'), strong:contains('Compensation')")
	if err == nil && salary != "" {
		job.SalaryRange = CleanText(salary)
	}

	// Extract job type
	jobType, err := ExtractText(page, "strong:contains('Type'), strong:contains('Employment')")
	if err == nil {
		job.JobType = CleanText(jobType)
	}

	// Extract posted date from page metadata
	posted, err := ExtractText(page, "time, [datetime], meta[property='article:published_time']")
	if err == nil {
		job.PostedDate = CleanText(posted)
	}

	// Extract requirements (look for bullet points or lists)
	requirements, err := ExtractText(page, "ul li, ol li")
	if err == nil && requirements != "" {
		reqList := strings.Split(requirements, "\n")
		// Filter out empty requirements
		filtered := make([]string, 0)
		for _, req := range reqList {
			cleaned := CleanText(req)
			if cleaned != "" {
				filtered = append(filtered, cleaned)
			}
		}
		job.Requirements = filtered
	}

	// Extract page ID from URL
	if strings.Contains(url, "-") {
		parts := strings.Split(url, "-")
		if len(parts) > 0 {
			jobID := parts[len(parts)-1]
			// Remove query params if any
			jobID = strings.Split(jobID, "?")[0]
			job.ExternalID = jobID
		}
	}

	// Extract HTML content
	html, err := page.Content()
	if err == nil {
		job.RawHTML = html
	}

	// Validate required fields
	if job.Title == "" {
		return nil, fmt.Errorf("missing required field: title")
	}

	// Notion pages may not have explicit company field, try to extract from domain
	if job.Company == "" {
		// Try to extract from subdomain (e.g., companyname.notion.site)
		if strings.Contains(url, ".notion.site") {
			parts := strings.Split(url, ".")
			if len(parts) > 0 {
				subdomain := strings.Replace(parts[0], "https://", "", 1)
				subdomain = strings.Replace(subdomain, "http://", "", 1)
				job.Company = CleanText(subdomain)
			}
		}
	}

	if job.Company == "" {
		job.Company = "Unknown Company (Notion Page)"
	}

	return job, nil
}

// GetSearchURLs returns Notion search URLs (not applicable)
func (p *NotionParser) GetSearchURLs(keywords, location string) []string {
	// Notion doesn't have a central job search
	return []string{}
}
