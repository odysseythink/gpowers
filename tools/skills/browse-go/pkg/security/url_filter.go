package security

import (
	"net/url"
	"regexp"
	"strings"
)

// ContentFilterResult is returned by each content filter.
type ContentFilterResult struct {
	Safe     bool
	Warnings []string
	Blocked  bool
	Message  string
}

// ContentFilter is a function that inspects content and returns warnings.
type ContentFilter func(content string, pageURL string, command string) ContentFilterResult

var registeredFilters []ContentFilter

// RegisterContentFilter adds a filter to the pipeline.
func RegisterContentFilter(filter ContentFilter) {
	registeredFilters = append(registeredFilters, filter)
}

// ClearContentFilters removes all registered filters (for testing).
func ClearContentFilters() {
	registeredFilters = nil
}

// FilterMode controls how filter warnings are handled.
type FilterMode string

const (
	FilterOff   FilterMode = "off"
	FilterWarn  FilterMode = "warn"
	FilterBlock FilterMode = "block"
)

// GetFilterMode returns the current filter mode from environment.
func GetFilterMode() FilterMode {
	mode := strings.ToLower(GetEnv("BROWSE_CONTENT_FILTER", ""))
	if mode == "off" || mode == "block" {
		return FilterMode(mode)
	}
	return FilterWarn // default
}

// RunContentFilters runs all registered filters against content.
func RunContentFilters(content string, pageURL string, command string) ContentFilterResult {
	mode := GetFilterMode()
	if mode == FilterOff {
		return ContentFilterResult{Safe: true}
	}

	var allWarnings []string
	blocked := false

	for _, filter := range registeredFilters {
		result := filter(content, pageURL, command)
		if !result.Safe {
			allWarnings = append(allWarnings, result.Warnings...)
			if mode == FilterBlock {
				blocked = true
			}
		}
	}

	if blocked && len(allWarnings) > 0 {
		return ContentFilterResult{
			Safe:     false,
			Warnings: allWarnings,
			Blocked:  true,
			Message:  "Content blocked: " + strings.Join(allWarnings, "; "),
		}
	}

	return ContentFilterResult{
		Safe:     len(allWarnings) == 0,
		Warnings: allWarnings,
	}
}

// Blocklist domains for exfiltration detection.
// Updating this set is a deliberate security decision.
// Every addition widens the attack surface (false positive risk).
var blocklistDomains = []string{
	"requestbin.com",
	"pipedream.com",
	"webhook.site",
	"hookbin.com",
	"requestcatcher.com",
	"burpcollaborator.net",
	"interact.sh",
	"canarytokens.com",
	"ngrok.io",
	"ngrok-free.app",
}

var urlInContentPattern = regexp.MustCompile(`https?://[^\s"'<>]+`)

// URLBlocklistFilter checks if the page URL or content contains blocklisted exfiltration domains.
func URLBlocklistFilter(content string, pageURL string, _command string) ContentFilterResult {
	var warnings []string

	// Check page URL
	if pageURL != "" {
		for _, domain := range blocklistDomains {
			if strings.Contains(pageURL, domain) {
				warnings = append(warnings, "Page URL matches blocklisted domain: "+domain)
			}
		}
	}

	// Check for blocklisted URLs in content
	contentUrls := urlInContentPattern.FindAllString(content, -1)
	for _, contentUrl := range contentUrls {
		u, err := url.Parse(contentUrl)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		for _, domain := range blocklistDomains {
			if host == domain || strings.HasSuffix(host, "."+domain) {
				warnings = append(warnings, "Content contains blocklisted URL: "+contentUrl)
				break
			}
		}
	}

	return ContentFilterResult{Safe: len(warnings) == 0, Warnings: warnings}
}

func init() {
	RegisterContentFilter(URLBlocklistFilter)
}
