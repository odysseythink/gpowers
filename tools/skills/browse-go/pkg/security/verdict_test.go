package security

import (
	"math"
	"testing"
)

func TestCombineVerdictSafe(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerRuleBased, Confidence: 0.1},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictSafe {
		t.Errorf("expected safe, got %s", result.Verdict)
	}
}

func TestCombineVerdictLogOnly(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerRuleBased, Confidence: 0.50},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictLogOnly {
		t.Errorf("expected log_only, got %s", result.Verdict)
	}
}

func TestCombineVerdictWarn(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerRuleBased, Confidence: 0.80},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictWarn {
		t.Errorf("expected warn, got %s", result.Verdict)
	}
}

func TestCombineVerdictBlockCanary(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerCanary, Confidence: 1.0},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictBlock {
		t.Errorf("expected block for canary leak, got %s", result.Verdict)
	}
	if result.Reason != "canary_leaked" {
		t.Errorf("expected canary_leaked reason, got %s", result.Reason)
	}
}

func TestCombineVerdictEnsembleBlock(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerRuleBased, Confidence: 0.80},
		{Layer: LayerAriaRegex, Confidence: 0.80},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictBlock {
		t.Errorf("expected block for ensemble agreement, got %s", result.Verdict)
	}
	if result.Reason != "ensemble_agreement" {
		t.Errorf("expected ensemble_agreement reason, got %s", result.Reason)
	}
}

func TestCombineVerdictSingleLayerHighDegradesToWarn(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerTestsavantContent, Confidence: 0.95},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictWarn {
		t.Errorf("expected warn for solo high content (SO-FP mitigation), got %s", result.Verdict)
	}
}

func TestCombineVerdictSingleLayerHighToolOutputBlock(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerTestsavantContent, Confidence: 0.95},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{ToolOutput: true})
	if result.Verdict != VerdictBlock {
		t.Errorf("expected block for tool-output solo high, got %s", result.Verdict)
	}
}

func TestCombineVerdictTranscriptBlockVote(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerTranscriptClassifier, Confidence: 0.90, Meta: map[string]interface{}{"verdict": "block"}},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	// Single transcript block with high confidence should degrade to warn
	if result.Verdict != VerdictWarn {
		t.Errorf("expected warn for solo transcript block, got %s", result.Verdict)
	}
}

func TestCombineVerdictTranscriptLowConfidenceBlock(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerTranscriptClassifier, Confidence: 0.20, Meta: map[string]interface{}{"verdict": "block"}},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	// Hallucination guard: low-confidence block drops to warn-vote only
	if result.Verdict == VerdictBlock {
		t.Errorf("expected not block for low-confidence transcript block (hallucination guard)")
	}
}

func TestCombineVerdictConfidenceMinForEnsemble(t *testing.T) {
	signals := []LayerSignal{
		{Layer: LayerRuleBased, Confidence: 0.99},
		{Layer: LayerAriaRegex, Confidence: 0.80},
	}
	result := CombineVerdict(signals, CombineVerdictOpts{})
	if result.Verdict != VerdictBlock {
		t.Errorf("expected block for ensemble agreement")
	}
	// Confidence should be min of contributing signals
	wantConf := 0.80
	if math.Abs(result.Confidence-wantConf) > 0.001 {
		t.Errorf("ensemble confidence = %f, want %f (min of contributors)", result.Confidence, wantConf)
	}
}
