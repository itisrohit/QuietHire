package parsers

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// LinkedInParser parses LinkedIn job listings
type LinkedInParser struct{}

func NewLinkedInParser() *LinkedInParser {
	return &LinkedInParser{}
}

func (p *LinkedInParser) CanParse(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return strings.Contains(parsedURL.Host, "linkedin.com")
}

func (p *LinkedInParser) Parse(htmlContent string, urlStr string) (*JobListing, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	listing := &JobListing{
		URL:      urlStr,
		Platform: "linkedin",
	}

	// Parse job title
	if titleNode := FindNodeByClass(doc, "top-card-layout__title"); titleNode != nil {
		listing.Title = CleanText(ExtractText(titleNode))
	}

	// Parse company name
	if companyNode := FindNodeByClass(doc, "topcard__org-name-link"); companyNode != nil {
		listing.Company = CleanText(ExtractText(companyNode))
	}

	// Parse location
	if locationNode := FindNodeByClass(doc, "topcard__flavor--bullet"); locationNode != nil {
		listing.Location = CleanText(ExtractText(locationNode))
	}

	// Parse job description
	if descNode := FindNodeByClass(doc, "show-more-less-html__markup"); descNode != nil {
		listing.Description = CleanText(ExtractText(descNode))
	}

	return listing, nil
}

func (p *LinkedInParser) GetSearchURLs(query, location string) []string {
	baseURL := "https://www.linkedin.com/jobs/search/"
	encodedQuery := url.QueryEscape(query)
	encodedLocation := url.QueryEscape(location)

	urls := []string{
		fmt.Sprintf("%s?keywords=%s&location=%s", baseURL, encodedQuery, encodedLocation),
		fmt.Sprintf("%s?keywords=%s&location=%s&f_TPR=r86400", baseURL, encodedQuery, encodedLocation), // Last 24 hours
	}

	return urls
}
