package security

import (
	"context"
	"regexp"
	"strings"
)

// compile regexes used across security package
var (
	scriptRegex      = regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	styleRegex       = regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	tagRegex         = regexp.MustCompile(`<[^>]+>`)
	whitespaceRegex  = regexp.MustCompile(`\s+`)
)

// RuleBasedClassifier is a lightweight, always-available classifier that uses
// pattern matching and heuristics. It serves as a fallback when ONNX models
// are unavailable and provides the aria_regex layer signal.
type RuleBasedClassifier struct {
	patterns []*regexp.Regexp
}

// NewRuleBasedClassifier creates the rule-based classifier with injection patterns.
func NewRuleBasedClassifier() *RuleBasedClassifier {
	return &RuleBasedClassifier{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ignore\s+(previous|above|all)\s+instructions?`),
			regexp.MustCompile(`(?i)you\s+are\s+(now|a)\s+`),
			regexp.MustCompile(`(?i)system\s*:\s*`),
			regexp.MustCompile(`(?i)\bdo\s+not\s+(follow|obey|listen)`),
			regexp.MustCompile(`(?i)\bexecute\s+(the\s+)?following`),
			regexp.MustCompile(`(?i)\bforget\s+(everything|all|your)`),
			regexp.MustCompile(`(?i)\bnew\s+instructions?\s*:`),
			regexp.MustCompile(`(?i)disregard\s+(previous|all|above)`),
			regexp.MustCompile(`(?i)act\s+as\s+(an?\s+)?(admin|root|developer|system)`),
			regexp.MustCompile(`(?i)override\s+(previous|all|default)\s+(instructions?|settings?|config)`),
			regexp.MustCompile(`(?i)from\s+now\s+on\s+you\s+(will|must|should)`),
			regexp.MustCompile(`(?i)your\s+(new|updated)\s+(instructions?|role|persona)`),
		},
	}
}

func (r *RuleBasedClassifier) Name() LayerName { return LayerRuleBased }

func (r *RuleBasedClassifier) Load(ctx context.Context) error { return nil }

func (r *RuleBasedClassifier) Status() string { return "ok" }

func (r *RuleBasedClassifier) Scan(ctx context.Context, text string) LayerSignal {
	if text == "" {
		return LayerSignal{Layer: LayerRuleBased, Confidence: 0}
	}

	plain := htmlToPlainText(text)
	lower := strings.ToLower(plain)

	// Count matches and track which patterns fired
	matchCount := 0
	var matchedPatterns []string
	for _, p := range r.patterns {
		if p.MatchString(lower) {
			matchCount++
			matchedPatterns = append(matchedPatterns, p.String()[:min(40, len(p.String()))])
		}
	}

	if matchCount == 0 {
		return LayerSignal{Layer: LayerRuleBased, Confidence: 0, Meta: map[string]interface{}{"patterns_matched": 0}}
	}

	// Confidence scales with number of distinct patterns matched.
	// 1 match -> 0.40 (LOG_ONLY), 2 -> 0.60, 3+ -> 0.75 (WARN threshold)
	confidence := 0.40 + float64(matchCount-1)*0.20
	if confidence > 0.85 {
		confidence = 0.85
	}

	return LayerSignal{
		Layer:      LayerRuleBased,
		Confidence: confidence,
		Meta: map[string]interface{}{
			"patterns_matched": matchCount,
			"matched":          matchedPatterns,
		},
	}
}

// ARIARegexClassifier detects prompt-injection patterns in ARIA labels.
// This is conceptually a separate layer from the rule-based classifier
// because it focuses on accessibility-tree attack vectors.
type ARIARegexClassifier struct {
	patterns []*regexp.Regexp
}

// NewARIARegexClassifier creates the ARIA-focused classifier.
func NewARIARegexClassifier() *ARIARegexClassifier {
	return &ARIARegexClassifier{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ignore\s+(previous|above|all)\s+instructions?`),
			regexp.MustCompile(`(?i)you\s+are\s+(now|a)\s+`),
			regexp.MustCompile(`(?i)system\s*:\s*`),
			regexp.MustCompile(`(?i)\bdo\s+not\s+(follow|obey|listen)`),
			regexp.MustCompile(`(?i)\bexecute\s+(the\s+)?following`),
			regexp.MustCompile(`(?i)\bforget\s+(everything|all|your)`),
			regexp.MustCompile(`(?i)\bnew\s+instructions?\s*:`),
		},
	}
}

func (a *ARIARegexClassifier) Name() LayerName { return LayerAriaRegex }

func (a *ARIARegexClassifier) Load(ctx context.Context) error { return nil }

func (a *ARIARegexClassifier) Status() string { return "ok" }

func (a *ARIARegexClassifier) Scan(ctx context.Context, text string) LayerSignal {
	if text == "" {
		return LayerSignal{Layer: LayerAriaRegex, Confidence: 0}
	}
	lower := strings.ToLower(text)
	matchCount := 0
	for _, p := range a.patterns {
		if p.MatchString(lower) {
			matchCount++
		}
	}
	if matchCount == 0 {
		return LayerSignal{Layer: LayerAriaRegex, Confidence: 0}
	}
	confidence := 0.45 + float64(matchCount-1)*0.25
	if confidence > 0.90 {
		confidence = 0.90
	}
	return LayerSignal{
		Layer:      LayerAriaRegex,
		Confidence: confidence,
		Meta:       map[string]interface{}{"patterns_matched": matchCount},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
