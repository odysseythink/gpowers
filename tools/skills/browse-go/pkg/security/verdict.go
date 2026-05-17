package security

import (
	"math"
)

// Thresholds for classifier output. Calibrated against BrowseSafe-Bench smoke
// (200 cases) + benign corpus (50 pages). BLOCK is intentionally conservative.
const (
	ThresholdBlock         = 0.85
	ThresholdWarn          = 0.75
	ThresholdLogOnly       = 0.40
	ThresholdSoloContentBlock = 0.92
)

// Verdict is the final security decision.
type Verdict string

const (
	VerdictSafe          Verdict = "safe"
	VerdictLogOnly       Verdict = "log_only"
	VerdictWarn          Verdict = "warn"
	VerdictBlock         Verdict = "block"
	VerdictUserOverrode  Verdict = "user_overrode"
)

// LayerName identifies which defense layer produced a signal.
type LayerName string

const (
	LayerTestsavantContent     LayerName = "testsavant_content"
	LayerDebertaContent        LayerName = "deberta_content"
	LayerTranscriptClassifier  LayerName = "transcript_classifier"
	LayerAriaRegex             LayerName = "aria_regex"
	LayerCanary                LayerName = "canary"
	LayerRuleBased             LayerName = "rule_based"
)

// LayerSignal is the output of a single defense layer.
type LayerSignal struct {
	Layer      LayerName
	Confidence float64
	Meta       map[string]interface{}
}

// SecurityResult is the combined verdict from all layers.
type SecurityResult struct {
	Verdict    Verdict
	Reason     string
	Signals    []LayerSignal
	Confidence float64
}

// SecurityStatus reflects the overall health of the security stack.
type SecurityStatus string

const (
	StatusProtected SecurityStatus = "protected"
	StatusDegraded  SecurityStatus = "degraded"
	StatusInactive  SecurityStatus = "inactive"
)

// StatusDetail is reported via /health for the shield icon.
type StatusDetail struct {
	Status      SecurityStatus
	Layers      map[string]string // layer -> "ok" | "degraded" | "off"
	LastUpdated string
}

// CombineVerdictOpts allows caller to pass context-specific options.
type CombineVerdictOpts struct {
	ToolOutput bool // if true, single-layer BLOCK kills session (tool outputs aren't user-authored)
}

type voteStrength string

const (
	voteBlock voteStrength = "block"
	voteWarn  voteStrength = "warn"
	voteNone  voteStrength = "none"
)

// classifyTranscript maps a transcript classifier signal to a vote strength.
// Uses label-first logic: Haiku's verdict label is the primary signal.
func classifyTranscript(signal LayerSignal) voteStrength {
	meta := signal.Meta
	if meta == nil {
		meta = map[string]interface{}{}
	}
	verdict, _ := meta["verdict"].(string)
	confidence := signal.Confidence

	if verdict == "block" {
		// Hallucination guard: verdict=block with confidence < LOG_ONLY drops to warn-vote.
		if confidence >= ThresholdLogOnly {
			return voteBlock
		}
		return voteWarn
	}
	if verdict == "warn" {
		return voteWarn
	}
	if verdict == "safe" {
		return voteNone
	}
	// Backward-compat: no meta.verdict. Confidence-only fallback.
	if confidence >= ThresholdWarn {
		return voteWarn
	}
	return voteNone
}

// CombineVerdict combines per-layer signals into a single verdict using the
// ensemble rule from the TS original.
//
// Ensemble rule (v1.5.2.0+):
//   - BLOCK requires 2 block-votes across testsavant + deberta + transcript
//   - Canary leak (confidence >= 1.0) always BLOCKs (deterministic)
//   - Tool-output branch: single-layer BLOCK kills session
//   - Content classifiers solo BLOCK threshold: 0.92 (higher than 0.85)
//   - Transcript solo BLOCK: meta.verdict == "block" AND confidence >= 0.85
func CombineVerdict(signals []LayerSignal, opts CombineVerdictOpts) SecurityResult {
	byLayerMax := make(map[LayerName]float64)
	var transcriptSignals []LayerSignal

	for _, s := range signals {
		if current, ok := byLayerMax[s.Layer]; !ok || s.Confidence > current {
			byLayerMax[s.Layer] = s.Confidence
		}
		if s.Layer == LayerTranscriptClassifier {
			transcriptSignals = append(transcriptSignals, s)
		}
	}

	content := byLayerMax[LayerTestsavantContent]
	deberta := byLayerMax[LayerDebertaContent]
	transcriptMax := byLayerMax[LayerTranscriptClassifier]
	canary := byLayerMax[LayerCanary]
	ruleBased := byLayerMax[LayerRuleBased]
	aria := byLayerMax[LayerAriaRegex]

	// Include rule-based and aria in the max ML calculation for solo block checks
	maxMl := math.Max(content, math.Max(deberta, math.Max(transcriptMax, math.Max(ruleBased, aria))))

	// Canary leak is deterministic. Never gated through ensemble.
	if canary >= 1.0 {
		return SecurityResult{
			Verdict:    VerdictBlock,
			Reason:     "canary_leaked",
			Signals:    signals,
			Confidence: 1.0,
		}
	}

	// Transcript vote: pick the strongest signal (block > warn > none).
	transcriptVote := voteNone
	for _, s := range transcriptSignals {
		v := classifyTranscript(s)
		if v == voteBlock {
			transcriptVote = voteBlock
			break
		}
		if v == voteWarn && transcriptVote != voteBlock {
			transcriptVote = voteWarn
		}
	}

	// Scalar-layer votes (content classifiers).
	contentBlockVote := content >= ThresholdWarn
	debertaBlockVote := deberta >= ThresholdWarn
	ruleBlockVote := ruleBased >= ThresholdWarn
	ariaBlockVote := aria >= ThresholdWarn

	blockVotes := 0
	if contentBlockVote {
		blockVotes++
	}
	if debertaBlockVote {
		blockVotes++
	}
	if ruleBlockVote {
		blockVotes++
	}
	if ariaBlockVote {
		blockVotes++
	}
	if transcriptVote == voteBlock {
		blockVotes++
	}

	// Ensemble: 2-of-N block-votes trigger BLOCK.
	if blockVotes >= 2 {
		contributing := []float64{}
		if contentBlockVote {
			contributing = append(contributing, content)
		}
		if debertaBlockVote {
			contributing = append(contributing, deberta)
		}
		if ruleBlockVote {
			contributing = append(contributing, ruleBased)
		}
		if ariaBlockVote {
			contributing = append(contributing, aria)
		}
		if transcriptVote == voteBlock {
			contributing = append(contributing, transcriptMax)
		}
		minConf := contributing[0]
		for _, c := range contributing[1:] {
			if c < minConf {
				minConf = c
			}
		}
		return SecurityResult{
			Verdict:    VerdictBlock,
			Reason:     "ensemble_agreement",
			Signals:    signals,
			Confidence: minConf,
		}
	}

	// Single-layer BLOCK checks.
	maxContentLayer := math.Max(content, deberta)
	maxContentLayer = math.Max(maxContentLayer, ruleBased)
	maxContentLayer = math.Max(maxContentLayer, aria)
	contentSoloBlock := maxContentLayer >= ThresholdSoloContentBlock
	transcriptSoloBlock := transcriptVote == voteBlock && transcriptMax >= ThresholdBlock
	singleLayerBlockReached := contentSoloBlock || transcriptSoloBlock

	if singleLayerBlockReached {
		if opts.ToolOutput {
			return SecurityResult{
				Verdict:    VerdictBlock,
				Reason:     "single_layer_tool_output",
				Signals:    signals,
				Confidence: maxMl,
			}
		}
		return SecurityResult{
			Verdict:    VerdictWarn,
			Reason:     "single_layer_high",
			Signals:    signals,
			Confidence: maxMl,
		}
	}

	if maxMl >= ThresholdWarn || transcriptVote == voteWarn {
		return SecurityResult{
			Verdict:    VerdictWarn,
			Reason:     "single_layer_medium",
			Signals:    signals,
			Confidence: maxMl,
		}
	}

	if maxMl >= ThresholdLogOnly {
		return SecurityResult{
			Verdict:    VerdictLogOnly,
			Signals:    signals,
			Confidence: maxMl,
		}
	}

	return SecurityResult{
		Verdict:    VerdictSafe,
		Signals:    signals,
		Confidence: maxMl,
	}
}
