package parsers

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// IndeedParser parses Indeed job listings
type IndeedParser struct{}

func NewIndeedParser() *IndeedParser {
	return &IndeedParser{}
}

func (p *IndeedParser) CanParse(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	return strings.Contains(parsedURL.Host, "indeed.com")
}

func (p *IndeedParser) Parse(htmlContent string, urlStr string) (*JobListing, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	listing := &JobListing{
		URL:      urlStr,
		Platform: "indeed",
	}

	// Parse job title
	if titleNode := FindNodeByClass(doc, "jobsearch-JobInfoHeader-title"); titleNode != nil {
		listing.Title = CleanText(ExtractText(titleNode))
	}

	// Parse company name
	if companyNode := FindNodeByClass(doc, "jobsearch-InlineCompanyRating"); companyNode != nil {
		listing.Company = CleanText(ExtractText(companyNode))
	}

	// Parse location
	if locationNode := FindNodeByClass(doc, "jobsearch-JobInfoHeader-subtitle"); locationNode != nil {
		locationText := ExtractText(locationNode)
		// Location is usually after the company name, separated by a dot or dash
		parts := strings.Split(locationText, "•")
		if len(parts) > 1 {
			listing.Location = CleanText(parts[1])
		}
	}

	// Parse job description
	if descNode := FindNodeByID(doc, "jobDescriptionText"); descNode != nil {
		listing.Description = CleanText(ExtractText(descNode))
	}

	return listing, nil
}

func (p *IndeedParser) GetSearchURLs(query, location string) []string {
	baseURL := "https://www.indeed.com/jobs"
	encodedQuery := url.QueryEscape(query)
	encodedLocation := url.QueryEscape(location)

	urls := []string{
		fmt.Sprintf("%s?q=%s&l=%s", baseURL, encodedQuery, encodedLocation),
		fmt.Sprintf("%s?q=%s&l=%s&sort=date", baseURL, encodedQuery, encodedLocation),
	}

	return urls
}
