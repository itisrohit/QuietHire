package parsers

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

// AshbyParser handles job parsing from Ashby ATS
type AshbyParser struct{}

// CanParse checks if URL is an Ashby job page
func (p *AshbyParser) CanParse(url string) bool {
	return strings.Contains(url, "jobs.ashbyhq.com") ||
		strings.Contains(url, ".ashbyhq.com/")
}

// Parse extracts job data from Ashby page
func (p *AshbyParser) Parse(page playwright.Page, url string) (*JobData, error) {
	job := &JobData{
		URL:    url,
		Source: "ashby",
	}

	// Extract job title
	title, err := ExtractText(page, "h1._jd_title_, h1[class*='title'], h1")
	if err == nil {
		job.Title = CleanText(title)
	}

	// Extract company name
	company, err := ExtractText(page, "div._jd_company_, [class*='company'], meta[property='og:site_name']")
	if err == nil {
		job.Company = CleanText(company)
	}

	// Extract location
	location, err := ExtractText(page, "div._jd_location_, [class*='location'], span[class*='location']")
	if err == nil {
		job.Location = CleanText(location)

		// Check if remote
		locationLower := strings.ToLower(location)
		if strings.Contains(locationLower, "remote") {
			job.Remote = true
		}
	}

	// Extract job description
	desc, err := ExtractText(page, "div._jd_description_, div[class*='description'], div[class*='content']")
	if err == nil {
		job.Description = CleanText(desc)
	}

	// Extract salary range
	salary, err := ExtractText(page, "div._jd_salary_, [class*='salary'], [class*='compensation']")
	if err == nil && salary != "" {
		job.SalaryRange = CleanText(salary)
	}

	// Extract job type
	jobType, err := ExtractText(page, "div._jd_type_, [class*='employment-type'], [class*='job-type']")
	if err == nil {
		job.JobType = CleanText(jobType)
	}

	// Extract posted date
	posted, err := ExtractText(page, "time, [datetime], [class*='posted']")
	if err == nil {
		job.PostedDate = CleanText(posted)
	}

	// Extract requirements
	requirements, err := ExtractText(page, "div._jd_requirements_, [class*='requirements'], ul li")
	if err == nil && requirements != "" {
		job.Requirements = strings.Split(requirements, "\n")
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

// GetSearchURLs returns Ashby search URLs (not typically used for direct search)
func (p *AshbyParser) GetSearchURLs(keywords, location string) []string {
	// Ashby is typically embedded in company career pages
	// Return empty as we don't have a central search URL
	return []string{}
}
