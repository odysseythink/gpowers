package security

import (
	"context"
	"fmt"
	"strings"
)

// Classifier scans text content and returns a LayerSignal indicating the
// likelihood of prompt injection or other malicious content.
type Classifier interface {
	// Name returns the classifier's layer name.
	Name() LayerName

	// Load initializes the classifier. May trigger model downloads.
	// Idempotent — safe to call multiple times.
	Load(ctx context.Context) error

	// Scan analyzes text and returns a signal.
	// On failure, returns confidence=0 with degraded meta (fail-open).
	Scan(ctx context.Context, text string) LayerSignal

	// Status returns the classifier's health.
	Status() string // "ok" | "degraded" | "off"
}

// MultiClassifier runs multiple classifiers and aggregates their signals.
type MultiClassifier struct {
	classifiers []Classifier
}

// NewMultiClassifier creates a multi-classifier from a list of classifiers.
func NewMultiClassifier(classifiers ...Classifier) *MultiClassifier {
	return &MultiClassifier{classifiers: classifiers}
}

// Load initializes all classifiers concurrently.
func (m *MultiClassifier) Load(ctx context.Context) error {
	var errs []error
	for _, c := range m.classifiers {
		if err := c.Load(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", c.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("classifier load errors: %v", errs)
	}
	return nil
}

// Scan runs all classifiers and returns all signals.
func (m *MultiClassifier) Scan(ctx context.Context, text string) []LayerSignal {
	var signals []LayerSignal
	for _, c := range m.classifiers {
		signals = append(signals, c.Scan(ctx, text))
	}
	return signals
}

// Status returns a map of classifier statuses.
func (m *MultiClassifier) Status() map[string]string {
	status := make(map[string]string)
	for _, c := range m.classifiers {
		status[string(c.Name())] = c.Status()
	}
	return status
}

// htmlToPlainText strips HTML tags and decodes entities.
// TestSavantAI was trained on plain text, not markup.
func htmlToPlainText(input string) string {
	if !strings.Contains(input, "<") && !strings.Contains(input, "&") {
		return input
	}
	// Drop script/style bodies entirely
	result := scriptRegex.ReplaceAllString(input, " ")
	result = styleRegex.ReplaceAllString(result, " ")
	// Drop remaining tags
	result = tagRegex.ReplaceAllString(result, " ")
	// Decode common entities
	result = strings.ReplaceAll(result, "&nbsp;", " ")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	// Collapse whitespace
	result = whitespaceRegex.ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}

// ScanPageContent is a convenience that normalizes HTML to plain text,
// truncates to 4000 chars, and runs all classifiers.
func (m *MultiClassifier) ScanPageContent(ctx context.Context, text string) []LayerSignal {
	if text == "" {
		return nil
	}
	plain := htmlToPlainText(text)
	if len(plain) > 4000 {
		plain = plain[:4000]
	}
	return m.Scan(ctx, plain)
}
