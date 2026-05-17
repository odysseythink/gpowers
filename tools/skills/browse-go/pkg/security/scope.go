package security

import (
	"fmt"
	"os"
	"strings"
)

// Scope defines a permission category for commands.
type Scope string

const (
	ScopeAll      Scope = "all"
	ScopeNavigate Scope = "navigate"
	ScopeRead     Scope = "read"
	ScopeInteract Scope = "interact"
	ScopeWrite    Scope = "write"
	ScopeInspect  Scope = "inspect"
	ScopeSystem   Scope = "system"
)

// commandScopes maps each command to its required scope(s).
// A command can have multiple scopes if it crosses categories.
var commandScopes = map[string][]Scope{
	// Navigation
	"goto":       {ScopeNavigate},
	"back":       {ScopeNavigate},
	"forward":    {ScopeNavigate},
	"reload":     {ScopeNavigate},
	"newtab":     {ScopeNavigate},
	"closetab":   {ScopeNavigate},
	"frame":      {ScopeNavigate},
	"connect":    {ScopeNavigate},
	"disconnect": {ScopeNavigate},

	// Reading
	"text":         {ScopeRead},
	"html":         {ScopeRead},
	"links":        {ScopeRead},
	"forms":        {ScopeRead},
	"accessibility":{ScopeRead},
	"data":         {ScopeRead},
	"media":        {ScopeRead},
	"inspect":      {ScopeRead},
	"snapshot":     {ScopeRead},
	"url":          {ScopeRead},
	"is":           {ScopeRead},

	// Interaction
	"click":  {ScopeInteract},
	"type":   {ScopeInteract},
	"press":  {ScopeInteract},
	"select": {ScopeInteract},
	"scroll": {ScopeInteract},
	"hover":  {ScopeInteract},
	"wait":   {ScopeInteract},
	"fill":   {ScopeInteract},
	"upload": {ScopeInteract},

	// Write / Script injection
	"write":     {ScopeWrite},
	"load-html": {ScopeWrite},
	"eval":      {ScopeWrite},
	"js":        {ScopeWrite},

	// Inspection / CDP
	"cdp":            {ScopeInspect},
	"console":        {ScopeInspect},
	"network":        {ScopeInspect},
	"dialog":         {ScopeInspect},
	"cookie":         {ScopeInspect},
	"cookies":        {ScopeInspect},
	"header":         {ScopeInspect},
	"storage":        {ScopeInspect},
	"style":          {ScopeInspect},
	"css":            {ScopeInspect},
	"attrs":          {ScopeInspect},
	"perf":           {ScopeInspect},
	"diff":           {ScopeInspect},
	"dialog-accept":  {ScopeInspect},
	"dialog-dismiss": {ScopeInspect},

	// System / Meta
	"status":           {ScopeSystem},
	"tabs":             {ScopeSystem},
	"tab":              {ScopeSystem},
	"tab-each":         {ScopeSystem},
	"help":             {ScopeSystem},
	"stop":             {ScopeSystem},
	"watch":            {ScopeSystem},
	"viewport":         {ScopeSystem},
	"responsive":       {ScopeSystem},
	"restart":          {ScopeSystem},
	"resume":           {ScopeSystem},
	"focus":            {ScopeSystem},
	"prettyscreenshot": {ScopeSystem},
	"screenshot":       {ScopeSystem},
	"pdf":              {ScopeSystem},
	"state":            {ScopeSystem},
	"chain":            {ScopeSystem},
	"cleanup":          {ScopeSystem},
	"archive":          {ScopeSystem},
	"download":         {ScopeSystem},
	"scrape":           {ScopeSystem},
	"ux-audit":         {ScopeSystem},
	"inbox":            {ScopeSystem},
	"useragent":        {ScopeSystem},

	// Domain skills
	"domain-skill":     {ScopeSystem},

	// Browser skills
	"skill":            {ScopeSystem},
}

// ScopeSet holds a set of granted scopes.
type ScopeSet struct {
	scopes map[Scope]bool
	all    bool
}

// NewScopeSet parses a scope string (comma-separated) into a set.
// If scopesStr is empty or "all", all scopes are granted.
func NewScopeSet(scopesStr string) *ScopeSet {
	ss := &ScopeSet{scopes: make(map[Scope]bool)}
	scopesStr = strings.TrimSpace(scopesStr)
	if scopesStr == "" || scopesStr == "all" {
		ss.all = true
		return ss
	}
	for _, s := range strings.Split(scopesStr, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "all" {
			ss.all = true
			return ss
		}
		switch Scope(s) {
		case ScopeNavigate, ScopeRead, ScopeInteract, ScopeWrite, ScopeInspect, ScopeSystem:
			ss.scopes[Scope(s)] = true
		}
	}
	return ss
}

// Has checks whether a specific scope is granted.
func (ss *ScopeSet) Has(scope Scope) bool {
	if ss == nil || ss.all {
		return true
	}
	return ss.scopes[scope]
}

// HasAny checks whether any of the given scopes is granted.
func (ss *ScopeSet) HasAny(scopes []Scope) bool {
	if ss == nil || ss.all {
		return true
	}
	for _, s := range scopes {
		if ss.scopes[s] {
			return true
		}
	}
	return false
}

// String returns the scope set as a comma-separated string.
func (ss *ScopeSet) String() string {
	if ss == nil || ss.all {
		return "all"
	}
	var parts []string
	for _, s := range []Scope{ScopeNavigate, ScopeRead, ScopeInteract, ScopeWrite, ScopeInspect, ScopeSystem} {
		if ss.scopes[s] {
			parts = append(parts, string(s))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ",")
}

// GetDefaultScopeSet returns the default scope set from environment.
// BROWSE_SCOPE controls the default scopes. Empty = all.
func GetDefaultScopeSet() *ScopeSet {
	return NewScopeSet(os.Getenv("BROWSE_SCOPE"))
}

// GetCommandScopes returns the required scopes for a command.
func GetCommandScopes(command string) []Scope {
	return commandScopes[command]
}

// CheckScope verifies that the granted scopes allow the given command.
// Returns nil if allowed, error otherwise.
func CheckScope(command string, granted *ScopeSet) error {
	required := GetCommandScopes(command)
	if len(required) == 0 {
		// Unknown commands: allow if "all" is granted, else deny
		if granted == nil || granted.all {
			return nil
		}
		return fmt.Errorf("command %q requires scope %q (not granted: %s)", command, ScopeAll, granted.String())
	}
	if granted.HasAny(required) {
		return nil
	}
	return fmt.Errorf("command %q requires scope %q (granted: %s)", command, required[0], granted.String())
}

// ScopeCategories returns all known scope categories.
func ScopeCategories() []Scope {
	return []Scope{ScopeNavigate, ScopeRead, ScopeInteract, ScopeWrite, ScopeInspect, ScopeSystem}
}
