package commands

// CdpScope controls which mutex tier is used.
type CdpScope string

const (
	CdpScopeTab     CdpScope = "tab"
	CdpScopeBrowser CdpScope = "browser"
)

// CdpOutput controls whether output is wrapped as untrusted.
type CdpOutput string

const (
	CdpOutputTrusted   CdpOutput = "trusted"
	CdpOutputUntrusted CdpOutput = "untrusted"
)

// CdpAllowEntry describes one allowed CDP method.
type CdpAllowEntry struct {
	Domain        string
	Method        string
	Scope         CdpScope
	Output        CdpOutput
	Justification string
}

// CDPAllowlist is the deny-default allowlist for CDP escape-hatch commands.
// Methods not on this list are rejected.
var CDPAllowlist = []CdpAllowEntry{
	// ─── Accessibility (read-only) ─────────────────────────────
	{Domain: "Accessibility", Method: "getFullAXTree", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Read-only AX tree extraction"},
	{Domain: "Accessibility", Method: "getPartialAXTree", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Read-only AX tree subtree by node"},
	{Domain: "Accessibility", Method: "getRootAXNode", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Read-only root AX node accessor"},
	// ─── DOM (read-only inspection) ────────────────────────────
	{Domain: "DOM", Method: "describeNode", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Inspect a DOM node by backend ID; pure read"},
	{Domain: "DOM", Method: "getBoxModel", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Pure geometric data (box dimensions); no content leak"},
	{Domain: "DOM", Method: "getNodeForLocation", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Pure coordinate→nodeId mapping; no content leak"},
	// ─── CSS (read-only) ───────────────────────────────────────
	{Domain: "CSS", Method: "getMatchedStylesForNode", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Read computed cascade for a node"},
	{Domain: "CSS", Method: "getComputedStyleForNode", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Computed style values are bounded"},
	{Domain: "CSS", Method: "getInlineStylesForNode", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Inline style content may contain attacker-controlled values"},
	// ─── Performance metrics ───────────────────────────────────
	{Domain: "Performance", Method: "getMetrics", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Pure numeric metrics"},
	{Domain: "Performance", Method: "enable", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Domain enable; required prerequisite"},
	{Domain: "Performance", Method: "disable", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Domain disable"},
	// ─── Tracing ───────────────────────────────────────────────
	{Domain: "Tracing", Method: "start", Scope: CdpScopeBrowser, Output: CdpOutputTrusted, Justification: "Trace category capture; browser-scoped"},
	{Domain: "Tracing", Method: "end", Scope: CdpScopeBrowser, Output: CdpOutputUntrusted, Justification: "Trace dump may contain URLs and page data"},
	// ─── Emulation ─────────────────────────────────────────────
	{Domain: "Emulation", Method: "setDeviceMetricsOverride", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Viewport/scale override"},
	{Domain: "Emulation", Method: "clearDeviceMetricsOverride", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Clear viewport override"},
	{Domain: "Emulation", Method: "setUserAgentOverride", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "UA override"},
	// ─── Page capture ──────────────────────────────────────────
	{Domain: "Page", Method: "captureScreenshot", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Screenshot bytes"},
	{Domain: "Page", Method: "printToPDF", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "PDF bytes"},
	// ─── Network metadata ──────────────────────────────────────
	{Domain: "Network", Method: "enable", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Domain enable; required prerequisite"},
	{Domain: "Network", Method: "disable", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Domain disable"},
	// ─── Target (frame debugging / multi-target inspection) ─────
	{Domain: "Target", Method: "getTargets", Scope: CdpScopeBrowser, Output: CdpOutputTrusted, Justification: "List all targets; read-only introspection"},
	{Domain: "Target", Method: "attachToTarget", Scope: CdpScopeBrowser, Output: CdpOutputTrusted, Justification: "Attach to an iframe or other target for debugging"},
	{Domain: "Target", Method: "detachFromTarget", Scope: CdpScopeBrowser, Output: CdpOutputTrusted, Justification: "Detach from a previously attached target"},
	{Domain: "Target", Method: "getTargetInfo", Scope: CdpScopeBrowser, Output: CdpOutputTrusted, Justification: "Read-only target metadata lookup"},
	// ─── Page (frame tree) ─────────────────────────────────────
	{Domain: "Page", Method: "getFrameTree", Scope: CdpScopeTab, Output: CdpOutputTrusted, Justification: "Read-only frame hierarchy introspection"},
	// ─── Runtime (limited, NO evaluate/callFunctionOn) ──────────
	{Domain: "Runtime", Method: "getProperties", Scope: CdpScopeTab, Output: CdpOutputUntrusted, Justification: "Inspect properties of an existing remote object"},
}

var cdpAllowlistIndex map[string]CdpAllowEntry

func init() {
	cdpAllowlistIndex = make(map[string]CdpAllowEntry, len(CDPAllowlist))
	for _, e := range CDPAllowlist {
		cdpAllowlistIndex[e.Domain+"."+e.Method] = e
	}
}

// LookupCdpMethod returns the allowlist entry for a qualified CDP method name,
// or nil if not allowed.
func LookupCdpMethod(qualified string) *CdpAllowEntry {
	if e, ok := cdpAllowlistIndex[qualified]; ok {
		return &e
	}
	return nil
}
