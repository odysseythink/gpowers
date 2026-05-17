// Package commands implements the browse command dispatcher.
package commands

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"browse-go/pkg/activity"
	"browse-go/pkg/browser"
	"browse-go/pkg/security"
)

// ExecContext holds the runtime context for a single command invocation.
type ExecContext struct {
	BM       *browser.BrowserManager
	Session  *browser.TabSession
	Args     []string
	ScopeSet *security.ScopeSet
	ClientID string
	Port     int // daemon listen port for skill spawn callbacks
}

// Handler is a command implementation.
type Handler func(ctx *ExecContext) (string, error)

// Registry maps canonical command names to their handlers.
type Registry struct {
	handlers     map[string]Handler
	descs        map[string]CommandDesc
	pipeline     *security.Pipeline
	rateLimiter  *security.RateLimiter
	failureMu    sync.Mutex
	failureCounts map[string]int // clientID -> consecutive failures
}

// CommandDesc provides metadata for a command.
type CommandDesc struct {
	Category    string
	Description string
	Usage       string
}

// NewRegistry creates a registry with all built-in commands registered.
func NewRegistry() *Registry {
	r := &Registry{
		handlers:      make(map[string]Handler),
		descs:         make(map[string]CommandDesc),
		pipeline:      security.NewPipeline(nil),
		rateLimiter:   security.NewRateLimiter(),
		failureCounts: make(map[string]int),
	}
	r.registerNavigation()
	r.registerTabs()
	r.registerServer()
	r.registerReading()
	r.registerInteraction()
	r.registerWrite()
	r.registerInspection()
	r.registerVisual()
	r.registerSnapshot()
	r.registerMeta()
	r.registerInbox()
	r.registerCdp()
	r.registerDomainSkill()
	r.registerBrowserSkill()
	return r
}

// Register adds a command handler and its description.
func (r *Registry) Register(name string, desc CommandDesc, h Handler) {
	r.handlers[name] = h
	r.descs[name] = desc
}

// Get looks up a handler by name. Aliases are resolved automatically.
func (r *Registry) Get(name string) (Handler, bool) {
	name = Canonicalize(name)
	h, ok := r.handlers[name]
	return h, ok
}

// GetDesc returns the description for a command.
func (r *Registry) GetDesc(name string) (CommandDesc, bool) {
	name = Canonicalize(name)
	d, ok := r.descs[name]
	return d, ok
}

// ListCommands returns all registered command names and descriptions,
// sorted by category then name.
func (r *Registry) ListCommands() []struct{ Name string; Desc CommandDesc } {
	out := make([]struct{ Name string; Desc CommandDesc }, 0, len(r.descs))
	for name, desc := range r.descs {
		out = append(out, struct{ Name string; Desc CommandDesc }{Name: name, Desc: desc})
	}
	// Sort by category, then name
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[i].Desc.Category > out[j].Desc.Category ||
				(out[i].Desc.Category == out[j].Desc.Category && out[i].Name > out[j].Name) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// Canonicalize normalizes a command name (lowercase, aliases resolved).
func Canonicalize(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "setcontent", "set_content":
		return "load-html"
	}
	return name
}

// noSessionCommands can run without an active browser tab.
var noSessionCommands = map[string]bool{
	"status": true, "tabs": true, "tab": true,
	"newtab": true, "closetab": true, "stop": true,
	"help": true, "inbox": true,
	"domain-skill": true, "skill": true,
	"cookie-import-browser": true,
}

// secureCommands receive security post-processing (L1-L6).
var secureCommands = map[string]bool{
	"text": true, "html": true, "links": true,
	"forms": true, "accessibility": true, "data": true,
	"media": true, "inspect": true, "snapshot": true,
	"download": true, "scrape": true,
}

// ExecuteOpts provides optional parameters for command execution.
type ExecuteOpts struct {
	ScopeSet *security.ScopeSet
	ClientID string
	Port     int // daemon listen port
}

// Execute runs a command by name against the active session.
// If opts is nil, default scope (all) and empty clientID are used.
func (r *Registry) Execute(bm *browser.BrowserManager, name string, args []string) (string, error) {
	return r.ExecuteWithOpts(bm, name, args, nil)
}

// ExecuteWithOpts runs a command with scope and rate limit checks.
func (r *Registry) ExecuteWithOpts(bm *browser.BrowserManager, name string, args []string, opts *ExecuteOpts) (string, error) {
	if opts == nil {
		opts = &ExecuteOpts{ScopeSet: security.NewScopeSet("")}
	}
	if opts.ScopeSet == nil {
		opts.ScopeSet = security.NewScopeSet("")
	}

	start := security.LogCommandStart(name, args, opts.ScopeSet.String(), opts.ClientID)
	var result *security.SecurityResult
	var execErr error
	var output string
	granted := true
	rateLimited := false

	// Capture page URL once (cached in BrowserManager to avoid round-trips)
	var pageURL string
	if bm != nil {
		pageURL = bm.CurrentURL()
	}

	defer func() {
		security.LogCommandEnd(start, name, args, security.ExtractDomain(pageURL),
			opts.ScopeSet.String(), opts.ClientID, granted, rateLimited, result, execErr)
	}()

	// 1. Look up command
	h, ok := r.Get(name)
	if !ok {
		execErr = fmt.Errorf("unknown command: %s", name)
		return "", execErr
	}

	// 2. Scope check
	canonName := Canonicalize(name)
	if err := security.CheckScope(canonName, opts.ScopeSet); err != nil {
		granted = false
		execErr = err
		return "", execErr
	}

	// 3. Rate limit check
	if r.rateLimiter != nil {
		key := security.KeyForCommand(opts.ClientID, canonName)
		if err := r.rateLimiter.Allow(key); err != nil {
			rateLimited = true
			execErr = err
			return "", execErr
		}
	}

	// 4. Get session
	var session *browser.TabSession
	if bm != nil {
		var err error
		session, err = bm.GetActiveSession()
		if err != nil && !noSessionCommands[name] {
			execErr = err
			return "", execErr
		}
	} else if !noSessionCommands[name] {
		execErr = fmt.Errorf("browser not initialized")
		return "", execErr
	}

	// 5. Execute handler
	activity.EmitCommandStart(name, args, pageURL)
	ctx := &ExecContext{BM: bm, Session: session, Args: args, ScopeSet: opts.ScopeSet, ClientID: opts.ClientID, Port: opts.Port}
	output, execErr = h(ctx)
	activity.EmitCommandEnd(name, args, pageURL, time.Since(start).Milliseconds(), execErr)

	// Track consecutive failures for auto-handoff hint
	r.failureMu.Lock()
	if execErr != nil {
		r.failureCounts[opts.ClientID]++
		if r.failureCounts[opts.ClientID] >= 3 {
			execErr = fmt.Errorf("%w\n\nHint: 3 consecutive failures. Consider using 'handoff' to let the user drive, then 'resume' to regain control.", execErr)
		}
	} else {
		delete(r.failureCounts, opts.ClientID)
	}
	r.failureMu.Unlock()

	if execErr != nil {
		return "", execErr
	}

	// 6. Security post-processing for commands that return external content.
	if r.pipeline != nil && secureCommands[name] && output != "" {
		var pageURL string
		if session != nil {
			pageURL = session.GetURL()
		}
		var secured string
		if name == "snapshot" {
			secured, result, _ = r.pipeline.SecureSnapshotResult(output, pageURL)
		} else {
			useDomStrip := false
			secured, result, _ = r.pipeline.SecureTextResult(nil, output, pageURL, name, useDomStrip)
		}
		output = secured
	}

	return output, nil
}

// RateLimiter returns the registry's rate limiter for status queries.
func (r *Registry) RateLimiter() *security.RateLimiter {
	return r.rateLimiter
}

// Pipeline returns the registry's security pipeline.
func (r *Registry) Pipeline() *security.Pipeline {
	return r.pipeline
}

// RateLimitStatus returns the current rate limit configuration.
func (r *Registry) RateLimitStatus() map[string]interface{} {
	if r.rateLimiter == nil {
		return map[string]interface{}{"enabled": false}
	}
	return r.rateLimiter.Status()
}
