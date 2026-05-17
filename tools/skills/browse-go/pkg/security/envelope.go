// Package security provides the prompt-injection defense stack.
//
// Layer 1: Content envelope wrapping — all untrusted external content is
// wrapped in a trust-boundary envelope to prevent prompt-boundary escape.
package security

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
)

const (
	// EnvelopeBegin marks the start of untrusted content.
	// Uses ══ (double horizontal line) to stand out from normal ASCII dashes.
	EnvelopeBegin = "═══ BEGIN UNTRUSTED WEB CONTENT ═══"
	// EnvelopeEnd marks the end of untrusted content.
	EnvelopeEnd = "═══ END UNTRUSTED WEB CONTENT ═══"
)

var (
	sessionMarker     string
	sessionMarkerOnce sync.Once
)

// ensureSessionMarker returns a per-process random marker for datamarking.
// Safe for concurrent use.
func ensureSessionMarker() string {
	sessionMarkerOnce.Do(func() {
		b := make([]byte, 3)
		if _, err := rand.Read(b); err == nil {
			sessionMarker = base64.StdEncoding.EncodeToString(b)[:4]
		} else {
			// Fallback — extremely unlikely
			sessionMarker = "MARK"
		}
	})
	return sessionMarker
}

// ResetSessionMarker resets the session marker (for testing).
func ResetSessionMarker() {
	sessionMarker = ""
	sessionMarkerOnce = sync.Once{}
}

// DatamarkContent inserts an invisible watermark into text content.
// The marker is embedded as a Unicode tag sequence after every 3rd
// sentence-ending period. This allows detection of exfiltration while
// keeping the text readable.
//
// Only applied to plain text output (not HTML, forms, or structured data).
func DatamarkContent(content string) string {
	marker := ensureSessionMarker()
	zwsp := "\u200B"
	taggedMarker := ""
	for _, c := range marker {
		taggedMarker += zwsp + string(c)
	}

	count := 0
	var result strings.Builder
	idx := 0
	for {
		j := strings.Index(content[idx:], ". ")
		if j == -1 {
			result.WriteString(content[idx:])
			break
		}
		j += idx
		result.WriteString(content[idx : j+2])
		count++
		if count%3 == 0 {
			result.WriteString(taggedMarker)
		}
		idx = j + 2
	}
	return result.String()
}

// EscapeEnvelopeSentinels defuses envelope markers that appear inside
// attacker-controlled content. Any raw BEGIN/END marker gets a zero-width
// space spliced through "CONTENT" so it still renders visibly but no longer
// matches the envelope grep the LLM anchors on.
//
// Both the wrap path (full-page content) and the split path (scoped snapshots)
// must funnel untrusted text through this helper before emitting the outer
// envelope, otherwise a page whose content contains the literal sentinel can
// close the envelope early and forge a fake "trusted" section.
func EscapeEnvelopeSentinels(content string) string {
	zwsp := "\u200B"
	// Page content envelope (═══)
	content = strings.ReplaceAll(content, EnvelopeBegin,
		"═══ BEGIN UNTRUSTED WEB C"+zwsp+"ONTENT ═══")
	content = strings.ReplaceAll(content, EnvelopeEnd,
		"═══ END UNTRUSTED WEB C"+zwsp+"ONTENT ═══")
	// External content envelope (---)
	content = strings.ReplaceAll(content, "--- BEGIN UNTRUSTED EXTERNAL CONTENT",
		"--- BEGIN UNTRUSTED EXTERNAL C"+zwsp+"ONTENT")
	content = strings.ReplaceAll(content, "--- END UNTRUSTED EXTERNAL CONTENT",
		"--- END UNTRUSTED EXTERNAL C"+zwsp+"ONTENT")
	return content
}

// WrapUntrustedPageContent wraps page content in a trust-boundary envelope.
// It first escapes any envelope sentinels inside the content, then wraps.
// Optional filter warnings are prepended as a banner.
func WrapUntrustedPageContent(content string, warnings []string) string {
	safeContent := EscapeEnvelopeSentinels(content)

	var parts []string
	if len(warnings) > 0 {
		parts = append(parts, "⚠ CONTENT WARNINGS: "+strings.Join(warnings, "; "))
	}
	parts = append(parts, EnvelopeBegin, safeContent, EnvelopeEnd)
	return strings.Join(parts, "\n")
}

// WrapUntrusted wraps arbitrary untrusted content in the standard envelope.
// This is the general-purpose version used by CDP, inbox, and other channels.
// The source parameter identifies the origin (e.g. "cdp:Accessibility.getFullAXTree").
func WrapUntrusted(content string, source string) string {
	safeContent := EscapeEnvelopeSentinels(content)
	// Sanitize source to prevent newline injection in the header
	safeSource := strings.ReplaceAll(source, "\n", "")
	if len(safeSource) > 200 {
		safeSource = safeSource[:200]
	}
	return "--- BEGIN UNTRUSTED EXTERNAL CONTENT (source: " + safeSource + ") ---\n" +
		safeContent + "\n--- END UNTRUSTED EXTERNAL CONTENT ---"
}
