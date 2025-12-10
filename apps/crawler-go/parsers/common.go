package parsers

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// JobListing represents a parsed job listing
type JobListing struct {
	Title       string
	Company     string
	Location    string
	Description string
	URL         string
	Platform    string
}

// Parser interface for all job board parsers
type Parser interface {
	CanParse(url string) bool
	Parse(htmlContent string, url string) (*JobListing, error)
	GetSearchURLs(query string, location string) []string
}

// ExtractText extracts all text from an HTML node
func ExtractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += ExtractText(c)
	}
	return text
}

// FindNodeByClass finds the first node with a specific class
func FindNodeByClass(n *html.Node, class string) *html.Node {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, class) {
				return n
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := FindNodeByClass(c, class); result != nil {
			return result
		}
	}

	return nil
}

// FindNodeByID finds a node by its ID
func FindNodeByID(n *html.Node, id string) *html.Node {
	if n.Type == html.ElementNode {
		for _, attr := range n.Attr {
			if attr.Key == "id" && attr.Val == id {
				return n
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := FindNodeByID(c, id); result != nil {
			return result
		}
	}

	return nil
}

// CleanText removes extra whitespace and newlines
func CleanText(text string) string {
	// Remove extra whitespace
	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
