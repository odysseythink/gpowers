package security

import (
	"context"
	"testing"
)

func TestPipeline_ScanTranscript_NotInitialized(t *testing.T) {
	p := NewPipeline(nil)
	sig := p.ScanTranscript(context.Background(), "hello", nil, "")
	if sig.Layer != LayerTranscriptClassifier {
		t.Errorf("expected layer %s, got %s", LayerTranscriptClassifier, sig.Layer)
	}
	if sig.Meta["degraded"] != true {
		t.Errorf("expected degraded for uninitialized haiku")
	}
}

func TestPipeline_Status_WithHaiku(t *testing.T) {
	p := NewPipeline(nil)
	status := p.Status()

	if status.Layers["transcript"] != "degraded" && status.Layers["transcript"] != "off" {
		t.Errorf("expected transcript off/degraded when haiku not loaded, got %s", status.Layers["transcript"])
	}
}

func TestPipeline_Load_InitializesHaiku(t *testing.T) {
	// Use a minimal classifier set to avoid slow model downloads in tests.
	mc := NewMultiClassifier(NewRuleBasedClassifier())
	p := NewPipeline(mc)
	// Load runs classifiers + haiku. Haiku will fail-open if claude CLI is missing.
	_ = p.Load(context.Background())
	// Should not panic and Status should be callable.
	_ = p.Status()
}
