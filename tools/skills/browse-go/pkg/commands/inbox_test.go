package commands

import (
	"strings"
	"testing"

	"browse-go/pkg/security"
)

func TestWrapUntrusted(t *testing.T) {
	result := security.WrapUntrusted("hello world", "test")
	if !strings.Contains(result, "BEGIN UNTRUSTED EXTERNAL CONTENT") {
		t.Errorf("expected UNTRUSTED wrapper, got: %s", result)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("expected content preserved, got: %s", result)
	}
	if !strings.Contains(result, "END UNTRUSTED EXTERNAL CONTENT") {
		t.Errorf("expected END marker, got: %s", result)
	}
}

func TestWrapUntrustedEscapesMarkers(t *testing.T) {
	result := security.WrapUntrusted("--- BEGIN UNTRUSTED EXTERNAL CONTENT", "test")
	// After escaping, the marker should contain a zero-width space (U+200B)
	if !strings.Contains(result, "C\u200BONTENT") {
		t.Errorf("marker injection should be escaped with zero-width space")
	}
}

func TestPlural(t *testing.T) {
	if plural(1) != "" {
		t.Errorf("plural(1) should be empty")
	}
	if plural(0) != "s" {
		t.Errorf("plural(0) should be 's'")
	}
	if plural(2) != "s" {
		t.Errorf("plural(2) should be 's'")
	}
}
