package commands

import (
	"testing"
)

func TestLookupCdpMethod(t *testing.T) {
	// Allowed methods
	for _, qualified := range []string{
		"Accessibility.getFullAXTree",
		"DOM.describeNode",
		"CSS.getMatchedStylesForNode",
		"Performance.getMetrics",
		"Page.captureScreenshot",
		"Network.enable",
		"Runtime.getProperties",
	} {
		entry := LookupCdpMethod(qualified)
		if entry == nil {
			t.Errorf("expected %s to be allowed", qualified)
		}
		if entry.Domain == "" || entry.Method == "" {
			t.Errorf("expected %s to have domain and method set", qualified)
		}
	}

	// Denied methods
	for _, qualified := range []string{
		"Runtime.evaluate",
		"Page.navigate",
		"Network.getResponseBody",
		"Target.createTarget",
		"Browser.close",
	} {
		entry := LookupCdpMethod(qualified)
		if entry != nil {
			t.Errorf("expected %s to be denied", qualified)
		}
	}
}

func TestCdpAllowEntryTags(t *testing.T) {
	for _, e := range CDPAllowlist {
		if e.Scope != CdpScopeTab && e.Scope != CdpScopeBrowser {
			t.Errorf("%s.%s: invalid scope %q", e.Domain, e.Method, e.Scope)
		}
		if e.Output != CdpOutputTrusted && e.Output != CdpOutputUntrusted {
			t.Errorf("%s.%s: invalid output %q", e.Domain, e.Method, e.Output)
		}
		if e.Justification == "" {
			t.Errorf("%s.%s: missing justification", e.Domain, e.Method)
		}
	}
}

func TestParseQualified(t *testing.T) {
	tests := []struct {
		input       string
		wantDomain  string
		wantMethod  string
		wantErr     bool
	}{
		{"Accessibility.getFullAXTree", "Accessibility", "getFullAXTree", false},
		{"Performance.getMetrics", "Performance", "getMetrics", false},
		{"invalid", "", "", true},
		{"nodot", "", "", true},
		{".", "", "", true},
		{"Domain.", "", "", true},
		{".method", "", "", true},
	}
	for _, tc := range tests {
		d, m, err := parseQualified(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseQualified(%q) expected error", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseQualified(%q) unexpected error: %v", tc.input, err)
			continue
		}
		if d != tc.wantDomain || m != tc.wantMethod {
			t.Errorf("parseQualified(%q) = (%q, %q), want (%q, %q)", tc.input, d, m, tc.wantDomain, tc.wantMethod)
		}
	}
}
