package parsers

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// GreenhouseParser parses Greenhouse ATS job listings
type GreenhouseParser struct{}

func NewGreenhouseParser() *GreenhouseParser {
	return &GreenhouseParser{}
}

func (p *GreenhouseParser) CanParse(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return strings.Contains(parsedURL.Host, "greenhouse.io") ||
		strings.Contains(parsedURL.Host, "boards.greenhouse.io")
}

func (p *GreenhouseParser) Parse(htmlContent string, urlStr string) (*JobListing, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	listing := &JobListing{
		URL:      urlStr,
		Platform: "greenhouse",
	}

	// Parse job title
	if titleNode := FindNodeByClass(doc, "app-title"); titleNode != nil {
		listing.Title = CleanText(ExtractText(titleNode))
	}

	// Parse company name (usually in the page header or metadata)
	if companyNode := FindNodeByClass(doc, "company-name"); companyNode != nil {
		listing.Company = CleanText(ExtractText(companyNode))
	}

	// Parse location
	if locationNode := FindNodeByClass(doc, "location"); locationNode != nil {
		listing.Location = CleanText(ExtractText(locationNode))
	}

	// Parse job description
	if descNode := FindNodeByID(doc, "content"); descNode != nil {
		listing.Description = CleanText(ExtractText(descNode))
	}

	return listing, nil
}

func (p *GreenhouseParser) GetSearchURLs(query, location string) []string {
	// Greenhouse doesn't have a universal search URL
	// Jobs are typically on company-specific boards
	// This would need to be customized per company
	return []string{}
}
