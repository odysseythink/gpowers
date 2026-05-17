// Package domainskill implements per-site notes the agent writes for itself.
//
// Storage:
//   - per-project: ~/.gstack/projects/<slug>/learnings.jsonl
//   - global:      ~/.gstack/global-domain-skills.jsonl
//
// State machine (T6):
//   quarantined ──(N=3 uses, no flags)──► active ──(manual promote)──► global
//
// Storage discipline:
//   - Append-only with O_APPEND (POSIX atomic appends < PIPE_BUF)
//   - Tombstone for deletes; compactor rewrites file
//   - Tolerant parser drops partial trailing line
package domainskill

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"browse-go/pkg/config"
	"browse-go/pkg/fs"
)

// ─── Types ──────────────────────────────────────────────────────

type State string

const (
	StateQuarantined State = "quarantined"
	StateActive      State = "active"
	StateGlobal      State = "global"
)

type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

// Row is a single domain-skill entry in JSONL storage.
type Row struct {
	Type            string  `json:"type"`
	Host            string  `json:"host"`
	Scope           Scope   `json:"scope"`
	State           State   `json:"state"`
	Body            string  `json:"body"`
	Version         int     `json:"version"`
	ClassifierScore float64 `json:"classifier_score"`
	Source          string  `json:"source"`
	SHA256          string  `json:"sha256"`
	UseCount        int     `json:"use_count"`
	FlagCount       int     `json:"flag_count"`
	CreatedTS       string  `json:"created_ts"`
	UpdatedTS       string  `json:"updated_ts"`
	Tombstone       bool    `json:"tombstone,omitempty"`
}

// ReadResult includes the source tier for a skill lookup.
type ReadResult struct {
	Row    Row
	Source Scope // 'project' or 'global'
}

// WriteInput is everything needed to save a new skill row.
type WriteInput struct {
	Host            string
	Body            string
	ProjectSlug     string
	Source          string // 'agent' or 'human'
	ClassifierScore float64
}

const promoteThreshold = 3

// ─── Hostname normalization (T3) ────────────────────────────────

// NormalizeHost strips protocol, path, port, and www. prefix.
func NormalizeHost(input string) string {
	h := strings.TrimSpace(strings.ToLower(input))
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimPrefix(h, "https://")
	if i := strings.IndexAny(h, "/?#"); i >= 0 {
		h = h[:i]
	}
	if i := strings.Index(h, ":"); i >= 0 {
		h = h[:i]
	}
	h = strings.TrimPrefix(h, "www.")
	return h
}

// DeriveHostFromURL extracts hostname from a page URL.
func DeriveHostFromURL(url string) (string, error) {
	if url == "" || url == "about:blank" || strings.HasPrefix(url, "chrome://") {
		return "", fmt.Errorf("cannot save domain-skill: no top-level URL on active tab\n" +
			"Cause: tab is empty or on chrome:// page.\n" +
			"Action: navigate to the target site first with $B goto <url>.")
	}
	return NormalizeHost(url), nil
}

// ─── File paths ─────────────────────────────────────────────────

func gstackHome() string {
	return config.Home()
}

func globalFile() string {
	return filepath.Join(gstackHome(), "global-domain-skills.jsonl")
}

func projectFile(slug string) string {
	return filepath.Join(gstackHome(), "projects", slug, "learnings.jsonl")
}

// ─── File I/O (append-only + atomic) ────────────────────────────

var appendMu sync.Mutex

func appendRow(filePath string, row Row) error {
	appendMu.Lock()
	defer appendMu.Unlock()

	if err := fs.MkdirSecure(filepath.Dir(filePath)); err != nil {
		return err
	}
	line, err := json.Marshal(row)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return err
	}
	return f.Sync()
}

func readRows(filePath string) ([]Row, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var rows []Row
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r Row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue // tolerate partial-line corruption
		}
		if r.Type == "domain" {
			rows = append(rows, r)
		}
	}
	return rows, nil
}

// ─── Latest-wins resolution ─────────────────────────────────────

type skillKey struct {
	Host  string
	Scope Scope
}

func keyOf(r Row) skillKey {
	return skillKey{Host: r.Host, Scope: r.Scope}
}

func resolveLatest(rows []Row) map[skillKey]Row {
	m := make(map[skillKey]Row)
	for _, r := range rows {
		k := keyOf(r)
		prior, ok := m[k]
		if !ok || r.Version >= prior.Version {
			m[k] = r
		}
	}
	// Drop tombstoned entries
	for k, r := range m {
		if r.Tombstone {
			delete(m, k)
		}
	}
	return m
}

// ─── Public API ─────────────────────────────────────────────────

// Read returns the active skill for a host visible to a project.
// Project-scoped active skills shadow global skills.
// Quarantined skills are never returned.
func Read(host, projectSlug string) (*ReadResult, error) {
	normalized := NormalizeHost(host)

	// Project layer first
	projectRows, err := readRows(projectFile(projectSlug))
	if err != nil {
		return nil, err
	}
	projectLatest := resolveLatest(projectRows)
	if hit, ok := projectLatest[skillKey{Host: normalized, Scope: ScopeProject}]; ok && hit.State == StateActive {
		return &ReadResult{Row: hit, Source: ScopeProject}, nil
	}

	// Global layer fallback
	globalRows, err := readRows(globalFile())
	if err != nil {
		return nil, err
	}
	globalLatest := resolveLatest(globalRows)
	if hit, ok := globalLatest[skillKey{Host: normalized, Scope: ScopeGlobal}]; ok && hit.State == StateGlobal {
		return &ReadResult{Row: hit, Source: ScopeGlobal}, nil
	}

	return nil, nil
}

// Write saves a new skill (always quarantined initially).
func Write(input WriteInput) (*Row, error) {
	if input.ClassifierScore >= 0.85 {
		return nil, fmt.Errorf("save blocked: classifier flagged content as potential injection (score: %.2f)\n"+
			"Cause: skill body contains patterns the L4 classifier marks as risky.\n"+
			"Action: rewrite the skill content removing instruction-like prose, retry.", input.ClassifierScore)
	}

	normalized := NormalizeHost(input.Host)
	body := input.Body
	now := time.Now().UTC().Format(time.RFC3339)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))

	// Determine prior version
	projectRows, _ := readRows(projectFile(input.ProjectSlug))
	projectLatest := resolveLatest(projectRows)
	prior, _ := projectLatest[skillKey{Host: normalized, Scope: ScopeProject}]
	version := 1
	createdTS := now
	if prior.Version > 0 {
		version = prior.Version + 1
		createdTS = prior.CreatedTS
	}

	row := Row{
		Type:            "domain",
		Host:            normalized,
		Scope:           ScopeProject,
		State:           StateQuarantined,
		Body:            body,
		Version:         version,
		ClassifierScore: input.ClassifierScore,
		Source:          input.Source,
		SHA256:          hash,
		UseCount:        0,
		FlagCount:       0,
		CreatedTS:       createdTS,
		UpdatedTS:       now,
	}
	if err := appendRow(projectFile(input.ProjectSlug), row); err != nil {
		return nil, err
	}
	return &row, nil
}

// RecordUse increments use_count and optionally promotes quarantined → active.
func RecordUse(host, projectSlug string, classifierFlagged bool) (*Row, error) {
	normalized := NormalizeHost(host)
	rows, err := readRows(projectFile(projectSlug))
	if err != nil {
		return nil, err
	}
	latest := resolveLatest(rows)
	current, ok := latest[skillKey{Host: normalized, Scope: ScopeProject}]
	if !ok {
		return nil, nil
	}

	useCount := current.UseCount + 1
	flagCount := current.FlagCount
	if classifierFlagged {
		flagCount++
	}
	state := current.State
	if state == StateQuarantined && useCount >= promoteThreshold && flagCount == 0 && current.ClassifierScore > 0 {
		state = StateActive
	}

	updated := Row{
		Type:            current.Type,
		Host:            current.Host,
		Scope:           current.Scope,
		State:           state,
		Body:            current.Body,
		Version:         current.Version + 1,
		ClassifierScore: current.ClassifierScore,
		Source:          current.Source,
		SHA256:          current.SHA256,
		UseCount:        useCount,
		FlagCount:       flagCount,
		CreatedTS:       current.CreatedTS,
		UpdatedTS:       time.Now().UTC().Format(time.RFC3339),
	}
	if err := appendRow(projectFile(projectSlug), updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// PromoteToGlobal promotes an active per-project skill to global scope.
func PromoteToGlobal(host, projectSlug string) (*Row, error) {
	normalized := NormalizeHost(host)
	rows, err := readRows(projectFile(projectSlug))
	if err != nil {
		return nil, err
	}
	latest := resolveLatest(rows)
	current, ok := latest[skillKey{Host: normalized, Scope: ScopeProject}]
	if !ok {
		return nil, fmt.Errorf("cannot promote: no skill for %s in project %s\n"+
			"Cause: skill does not exist or is tombstoned.\n"+
			"Action: $B domain-skill list to see what exists in this project.", normalized, projectSlug)
	}
	if current.State != StateActive {
		return nil, fmt.Errorf("cannot promote: skill for %s is in state %q, expected %q\n"+
			"Cause: skill must be active in this project (used %d+ times without flag) before global promotion.\n"+
			"Action: use the skill in this project until it auto-promotes to active.",
			normalized, current.State, StateActive, promoteThreshold)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	globalRow := Row{
		Type:            current.Type,
		Host:            current.Host,
		Scope:           ScopeGlobal,
		State:           StateGlobal,
		Body:            current.Body,
		Version:         1,
		ClassifierScore: current.ClassifierScore,
		Source:          current.Source,
		SHA256:          current.SHA256,
		UseCount:        0,
		FlagCount:       0,
		CreatedTS:       current.CreatedTS,
		UpdatedTS:       now,
	}
	if err := appendRow(globalFile(), globalRow); err != nil {
		return nil, err
	}
	return &globalRow, nil
}

// Rollback restores a prior version by re-emitting it as the latest.
func Rollback(host, projectSlug string, scope Scope) (*Row, error) {
	normalized := NormalizeHost(host)
	file := projectFile(projectSlug)
	if scope == ScopeGlobal {
		file = globalFile()
	}
	rows, err := readRows(file)
	if err != nil {
		return nil, err
	}
	var matching []Row
	for _, r := range rows {
		if r.Host == normalized && r.Scope == scope && !r.Tombstone {
			matching = append(matching, r)
		}
	}
	if len(matching) < 2 {
		return nil, fmt.Errorf("cannot rollback: %s has fewer than 2 versions in %s scope\n"+
			"Cause: no prior version to roll back to.\n"+
			"Action: $B domain-skill rm to delete instead, or wait for a future revision to roll back from.",
			normalized, scope)
	}

	// Sort by version desc; take second-latest
	for i := 0; i < len(matching)-1; i++ {
		for j := i + 1; j < len(matching); j++ {
			if matching[j].Version > matching[i].Version {
				matching[i], matching[j] = matching[j], matching[i]
			}
		}
	}
	target := matching[1]
	newVersion := matching[0].Version + 1

	restored := Row{
		Type:            target.Type,
		Host:            target.Host,
		Scope:           target.Scope,
		State:           target.State,
		Body:            target.Body,
		Version:         newVersion,
		ClassifierScore: target.ClassifierScore,
		Source:          target.Source,
		SHA256:          target.SHA256,
		UseCount:        target.UseCount,
		FlagCount:       target.FlagCount,
		CreatedTS:       target.CreatedTS,
		UpdatedTS:       time.Now().UTC().Format(time.RFC3339),
	}
	if err := appendRow(file, restored); err != nil {
		return nil, err
	}
	return &restored, nil
}

// ListResult groups skills by tier.
type ListResult struct {
	Project []Row
	Global  []Row
}

// List returns all non-tombstoned skills visible to a project.
func List(projectSlug string) (*ListResult, error) {
	projectRows, err := readRows(projectFile(projectSlug))
	if err != nil {
		return nil, err
	}
	globalRows, err := readRows(globalFile())
	if err != nil {
		return nil, err
	}
	projectLatest := resolveLatest(projectRows)
	globalLatest := resolveLatest(globalRows)

	var pRows []Row
	for _, r := range projectLatest {
		pRows = append(pRows, r)
	}
	var gRows []Row
	for _, r := range globalLatest {
		if r.State == StateGlobal {
			gRows = append(gRows, r)
		}
	}
	return &ListResult{Project: pRows, Global: gRows}, nil
}

// Delete tombstones a skill (appends tombstone row).
func Delete(host, projectSlug string, scope Scope) error {
	normalized := NormalizeHost(host)
	file := projectFile(projectSlug)
	if scope == ScopeGlobal {
		file = globalFile()
	}
	rows, err := readRows(file)
	if err != nil {
		return err
	}
	latest := resolveLatest(rows)
	current, ok := latest[skillKey{Host: normalized, Scope: scope}]
	if !ok {
		return fmt.Errorf("cannot delete: no skill for %s in %s scope\n"+
			"Cause: skill does not exist or is already tombstoned.\n"+
			"Action: $B domain-skill list to see what exists.", normalized, scope)
	}

	tombstone := Row{
		Type:            current.Type,
		Host:            current.Host,
		Scope:           current.Scope,
		State:           current.State,
		Body:            current.Body,
		Version:         current.Version + 1,
		ClassifierScore: current.ClassifierScore,
		Source:          current.Source,
		SHA256:          current.SHA256,
		UseCount:        current.UseCount,
		FlagCount:       current.FlagCount,
		CreatedTS:       current.CreatedTS,
		UpdatedTS:       time.Now().UTC().Format(time.RFC3339),
		Tombstone:       true,
	}
	return appendRow(file, tombstone)
}

// Compactor rewrites a JSONL file, dropping tombstoned and superseded rows.
func Compactor(filePath string) error {
	rows, err := readRows(filePath)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	latest := resolveLatest(rows)
	if len(latest) == 0 {
		_ = os.Remove(filePath)
		return nil
	}

	// Write compacted file
	tmpPath := filePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	for _, r := range latest {
		line, _ := json.Marshal(r)
		f.Write(line)
		f.Write([]byte{'\n'})
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, filePath)
}
