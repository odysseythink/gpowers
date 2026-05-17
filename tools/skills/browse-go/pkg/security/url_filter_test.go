package security

import (
	"testing"
)

func TestURLBlocklistFilter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		pageURL string
		wantSafe bool
	}{
		{
			name:     "safe page",
			content:  "hello world",
			pageURL:  "https://example.com",
			wantSafe: true,
		},
		{
			name:     "blocklisted page URL",
			content:  "hello world",
			pageURL:  "https://webhook.site/abc123",
			wantSafe: false,
		},
		{
			name:     "blocklisted URL in content",
			content:  "visit https://ngrok.io/evil for more",
			pageURL:  "https://example.com",
			wantSafe: false,
		},
		{
			name:     "blocklisted subdomain",
			content:  "link: https://evil.ngrok-free.app/data",
			pageURL:  "https://example.com",
			wantSafe: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := URLBlocklistFilter(tt.content, tt.pageURL, "test")
			if result.Safe != tt.wantSafe {
				t.Errorf("URLBlocklistFilter(%q, %q).Safe = %v, want %v", tt.content, tt.pageURL, result.Safe, tt.wantSafe)
			}
		})
	}
}

func TestRunContentFilters(t *testing.T) {
	// Ensure filter is registered
	if len(registeredFilters) == 0 {
		t.Fatal("no filters registered")
	}

	result := RunContentFilters("visit https://webhook.site", "https://example.com", "test")
	if result.Safe {
		t.Errorf("expected unsafe for webhook.site content")
	}
	if len(result.Warnings) == 0 {
		t.Errorf("expected warnings")
	}
}

func TestRunContentFiltersOff(t *testing.T) {
	t.Setenv("BROWSE_CONTENT_FILTER", "off")
	result := RunContentFilters("visit https://webhook.site", "https://example.com", "test")
	if !result.Safe {
		t.Errorf("expected safe when filter mode is off")
	}
}
