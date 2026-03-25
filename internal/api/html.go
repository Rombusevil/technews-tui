package api

import (
	"html"
	"regexp"
	"strings"
)

var (
	reLink      = regexp.MustCompile(`(?s)<a\s+href="([^"]*)"[^>]*>(.*?)</a>`)
	reCode      = regexp.MustCompile(`(?s)<code>(.*?)</code>`)
	reParagraph = regexp.MustCompile(`(?i)<p>`)
	reHTMLTag   = regexp.MustCompile(`<[^>]*>`)
)

// StripHTML converts HN HTML text to plain text for terminal display.
func StripHTML(s string) string {
	if s == "" {
		return ""
	}

	// Decode HTML entities first (handles &#x2F; etc.)
	s = html.UnescapeString(s)

	// Convert <code> to backtick-wrapped text
	s = reCode.ReplaceAllString(s, "`$1`")

	// Convert <a href="...">text</a> to "text (url)"
	s = reLink.ReplaceAllString(s, "$2 ($1)")

	// Convert <p> to double newlines
	s = reParagraph.ReplaceAllString(s, "\n\n")

	// Strip remaining HTML tags
	s = reHTMLTag.ReplaceAllString(s, "")

	// Clean up leading/trailing whitespace
	s = strings.TrimSpace(s)

	return s
}
