package security

import (
	"context"
	"fmt"
)

// Pipeline is the top-level security orchestrator. It wires together all
// layers (L1-L6) and provides a single entry point for securing command output.
type Pipeline struct {
	classifiers        *MultiClassifier
	haiku              *HaikuClassifier
	canary             string
}

// NewPipeline creates a security pipeline with the given classifiers.
// If classifiers is nil, a default set (rule-based + ARIA + ONNX) is used.
// Haiku transcript classifier is always instantiated but kept separate from
// the page-content MultiClassifier because it is slow (API round-trip) and
// only meaningful for transcript-shaped input (user message + tool calls).
func NewPipeline(classifiers *MultiClassifier) *Pipeline {
	if classifiers == nil {
		classifiers = NewMultiClassifier(
			NewRuleBasedClassifier(),
			NewARIARegexClassifier(),
			NewTestSavantClassifier(),
		)
		if isDebertaEnabled() {
			classifiers = NewMultiClassifier(
				NewRuleBasedClassifier(),
				NewARIARegexClassifier(),
				NewTestSavantClassifier(),
				NewDebertaClassifier(),
			)
		}
	}
	return &Pipeline{classifiers: classifiers, haiku: NewHaikuClassifier()}
}

// Load initializes all classifiers (content + Haiku).
func (p *Pipeline) Load(ctx context.Context) error {
	var errs []error
	if err := p.classifiers.Load(ctx); err != nil {
		errs = append(errs, err)
	}
	if p.haiku != nil {
		if err := p.haiku.Load(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("pipeline load errors: %v", errs)
	}
	return nil
}

// SetCanary sets the session canary token.
func (p *Pipeline) SetCanary(canary string) {
	p.canary = canary
}

// SecureTextResult applies the full security pipeline to a text result.
// It runs L2 (if applicable), L3, L4, L5, L6, and L1 (envelope wrapping).
//
// Parameters:
//   - ctx: browser tab context (for JS execution, can be nil)
//   - content: the raw command output
//   - pageURL: the current page URL (for blocklist and logging)
//   - command: the command name (for filter routing)
//   - useDomStrip: whether to run L2 hidden element stripping
func (p *Pipeline) SecureTextResult(ctx context.Context, content string, pageURL string, command string, useDomStrip bool) (string, *SecurityResult, error) {
	if content == "" {
		return content, nil, nil
	}

	var warnings []string

	// L2: Hidden element stripping (if we have a browser context)
	if useDomStrip && ctx != nil {
		found, err := MarkHiddenElements(ctx)
		if err == nil && len(found) > 0 {
			warnings = append(warnings, fmt.Sprintf("%d hidden elements flagged", len(found)))
			// Get clean text with hidden elements stripped
			cleanText, err := GetCleanText(ctx)
			if err == nil && cleanText != "" {
				content = cleanText
			}
			_ = CleanupHiddenMarkers(ctx)
		}
	}

	// L3: Content filters (URL blocklist)
	filterResult := RunContentFilters(content, pageURL, command)
	if !filterResult.Safe {
		warnings = append(warnings, filterResult.Warnings...)
		if filterResult.Blocked {
			return WrapUntrustedPageContent(filterResult.Message, warnings),
				&SecurityResult{Verdict: VerdictBlock, Reason: "content_filter", Signals: nil, Confidence: 1.0},
				nil
		}
	}

	// L4: ML classifier scan
	var signals []LayerSignal
	if p.classifiers != nil {
		signals = p.classifiers.ScanPageContent(context.Background(), content)
	}

	// L5: Canary check
	if p.canary != "" && CheckCanaryInString(content, p.canary) {
		signals = append(signals, LayerSignal{Layer: LayerCanary, Confidence: 1.0})
	}

	// L6: Combine verdict
	result := CombineVerdict(signals, CombineVerdictOpts{})

	// L1a: Datamarking (only for text command output)
	if command == "text" {
		content = DatamarkContent(content)
	}

	// L1: Envelope wrapping
	wrapped := WrapUntrustedPageContent(content, warnings)

	// Log attempts
	if result.Verdict == VerdictBlock || result.Verdict == VerdictWarn {
		_ = LogAttempt(AttemptRecord{
			URLDomain:   ExtractDomain(pageURL),
			PayloadHash: HashPayload(content),
			Confidence:  result.Confidence,
			Layer:       string(result.Reason),
			Verdict:     string(result.Verdict),
		})
	}

	return wrapped, &result, nil
}

// SecureSnapshotResult applies security to a snapshot/HTML result.
func (p *Pipeline) SecureSnapshotResult(content string, pageURL string) (string, *SecurityResult, error) {
	if content == "" {
		return content, nil, nil
	}

	// L3: URL blocklist
	filterResult := RunContentFilters(content, pageURL, "snapshot")
	var warnings []string
	if !filterResult.Safe {
		warnings = append(warnings, filterResult.Warnings...)
	}

	// L4: ML scan
	var signals []LayerSignal
	if p.classifiers != nil {
		signals = p.classifiers.ScanPageContent(context.Background(), content)
	}

	// L5: Canary
	if p.canary != "" && CheckCanaryInString(content, p.canary) {
		signals = append(signals, LayerSignal{Layer: LayerCanary, Confidence: 1.0})
	}

	result := CombineVerdict(signals, CombineVerdictOpts{})
	wrapped := WrapUntrustedPageContent(content, warnings)

	if result.Verdict == VerdictBlock || result.Verdict == VerdictWarn {
		_ = LogAttempt(AttemptRecord{
			URLDomain:   ExtractDomain(pageURL),
			PayloadHash: HashPayload(content),
			Confidence:  result.Confidence,
			Layer:       string(result.Reason),
			Verdict:     string(result.Verdict),
		})
	}

	return wrapped, &result, nil
}

// SecureCDPResult wraps CDP output in the untrusted envelope.
func (p *Pipeline) SecureCDPResult(content string, source string) string {
	return WrapUntrusted(content, source)
}

// SecureInboxResult wraps inbox output in the untrusted envelope.
func (p *Pipeline) SecureInboxResult(content string, source string) string {
	return WrapUntrusted(content, source)
}

// ScanTranscript runs the Haiku transcript classifier on a user message + tool
// calls. This is the L4b path from the TypeScript original. It is separate from
// page-content scanning because Haiku is slow (API round-trip) and only
// meaningful for transcript-shaped input.
//
// Gating: callers SHOULD only invoke when another layer already fired at >=
// ThresholdLogOnly. Skipping clean calls saves ~70%% of Haiku spend.
func (p *Pipeline) ScanTranscript(ctx context.Context, userMessage string, toolCalls []ToolCallInput, toolOutput string) LayerSignal {
	if p.haiku == nil {
		return LayerSignal{Layer: LayerTranscriptClassifier, Confidence: 0, Meta: map[string]interface{}{"degraded": true, "reason": "haiku_not_initialized"}}
	}
	return p.haiku.ScanTranscript(ctx, userMessage, toolCalls, toolOutput)
}

// Status returns the current security status for the shield icon.
func (p *Pipeline) Status() *StatusDetail {
	detail := GetStatus()
	if p.haiku != nil {
		detail.Layers["transcript"] = p.haiku.Status()
	}
	// Recompute overall status with transcript layer
	if detail.Layers["testsavant"] == "ok" && detail.Layers["transcript"] == "ok" && detail.Layers["canary"] == "ok" {
		detail.Status = StatusProtected
	} else if detail.Layers["testsavant"] == "off" && detail.Layers["canary"] == "off" {
		detail.Status = StatusInactive
	} else {
		detail.Status = StatusDegraded
	}
	return detail
}
