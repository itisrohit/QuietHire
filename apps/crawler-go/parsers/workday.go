package parsers

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// WorkdayParser handles job parsing from Workday
type WorkdayParser struct{}

// CanParse checks if URL is a Workday job page
func (p *WorkdayParser) CanParse(url string) bool {
	return strings.Contains(url, ".myworkdayjobs.com") ||
		strings.Contains(url, "workday.com")
}

// Parse extracts job data from Workday page
func (p *WorkdayParser) Parse(page playwright.Page, url string) (*JobData, error) {
	job := &JobData{
		URL:    url,
		Source: "workday",
	}

	// Extract job title
	title, err := ExtractText(page, "h2[data-automation-id='jobPostingHeader'], h1[class*='title'], h2")
	if err == nil {
		job.Title = CleanText(title)
	}

	// Extract company name (often in URL or site metadata)
	company, err := ExtractText(page, "[data-automation-id='company'], meta[property='og:site_name']")
	if err == nil {
		job.Company = CleanText(company)
	}

	// Fallback: extract company from URL (e.g., companyname.myworkdayjobs.com)
	if job.Company == "" {
		parts := strings.Split(url, ".")
		if len(parts) > 0 {
			company := strings.Split(parts[0], "//")
			if len(company) > 1 {
				job.Company = CleanText(company[1])
			}
		}
	}

	// Extract location
	location, err := ExtractText(page, "[data-automation-id='locations'], [class*='location']")
	if err == nil {
		job.Location = CleanText(location)

		// Check if remote
		locationLower := strings.ToLower(location)
		if strings.Contains(locationLower, "remote") {
			job.Remote = true
		}
	}

	// Extract job description
	desc, err := ExtractText(page, "[data-automation-id='jobPostingDescription'], div[class*='description']")
	if err == nil {
		job.Description = CleanText(desc)
	}

	// Extract job type
	jobType, err := ExtractText(page, "[data-automation-id='jobType'], [data-automation-id='time-type']")
	if err == nil {
		job.JobType = CleanText(jobType)
	}

	// Extract posted date
	posted, err := ExtractText(page, "[data-automation-id='postedOn'], time")
	if err == nil {
		job.PostedDate = CleanText(posted)
	}

	// Extract requirements
	requirements, err := ExtractText(page, "[data-automation-id='qualifications'], div[class*='qualifications'] li")
	if err == nil && requirements != "" {
		job.Requirements = strings.Split(requirements, "\n")
	}

	// Extract job ID from URL or page
	if strings.Contains(url, "/job/") {
		parts := strings.Split(url, "/job/")
		if len(parts) > 1 {
			jobID := strings.Split(parts[1], "/")[0]
			job.ExternalID = jobID
		}
	}

	// Extract HTML content
	html, err := page.Content()
	if err == nil {
		job.RawHTML = html
	}

	// Validate required fields
	if job.Title == "" || job.Company == "" {
		return nil, fmt.Errorf("missing required fields: title=%q, company=%q", job.Title, job.Company)
	}

	return job, nil
}

// GetSearchURLs returns Workday search URLs (not typically used for direct search)
func (p *WorkdayParser) GetSearchURLs(keywords, location string) []string {
	// Workday is company-specific (e.g., amazon.myworkdayjobs.com)
	// Return empty as we don't have a central search URL
	return []string{}
}
