package browser

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTabSessionRefMap(t *testing.T) {
	// Use background context — no real browser needed for ref map ops
	ts := NewTabSession(context.Background())
	defer ts.Close()

	// Set refs
	refs := map[string]RefEntry{
		"e1": {Selector: "button#submit", Role: "button", Name: "Submit"},
		"c1": {Selector: "input#name",     Role: "textbox", Name: "Name"},
	}
	ts.SetRefMap(refs)

	if ts.RefCount() != 2 {
		t.Fatalf("expected 2 refs, got %d", ts.RefCount())
	}

	entries := ts.RefEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Resolve non-ref selector returns unchanged
	sel, err := ts.ResolveRef("#foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sel != "#foo" {
		t.Fatalf("expected '#foo', got %s", sel)
	}

	// Resolve unknown ref returns error (no DOM lookup needed)
	_, err = ts.ResolveRef("@e99")
	if err == nil {
		t.Fatal("expected error for unknown ref")
	}

	// GetRefRole
	if ts.GetRefRole("@e1") != "button" {
		t.Fatalf("expected role 'button', got %s", ts.GetRefRole("@e1"))
	}
	if ts.GetRefRole("#foo") != "" {
		t.Fatal("expected empty role for non-ref")
	}

	// Clear
	ts.ClearRefs()
	if ts.RefCount() != 0 {
		t.Fatalf("expected 0 refs after clear, got %d", ts.RefCount())
	}
}

func TestTabSessionResolveRefStaleNoBrowser(t *testing.T) {
	// Without a real browser, ResolveRef on a known ref will fail at the
	// DOM lookup stage. Verify it returns a stale error.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ts := NewTabSession(ctx)
	defer ts.Close()

	ts.SetRefMap(map[string]RefEntry{
		"e1": {Selector: "button#x", Role: "button", Name: "X"},
	})

	_, err := ts.ResolveRef("@e1")
	if err == nil {
		t.Fatal("expected stale error without browser")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("expected 'stale' in error, got: %v", err)
	}
}

func TestTabSessionSnapshotAndHTML(t *testing.T) {
	ts := NewTabSession(context.Background())
	defer ts.Close()

	// Snapshot persists across nav
	ts.SetLastSnapshot("old snapshot")
	if ts.GetLastSnapshot() != "old snapshot" {
		t.Fatal("snapshot mismatch")
	}

	// Loaded HTML
	ts.SetTabContent("<html>hello</html>", "domcontentloaded")
	html, wait, ok := ts.GetLoadedHtml()
	if !ok || html != "<html>hello</html>" || wait != "domcontentloaded" {
		t.Fatalf("loaded html mismatch: %q %q %v", html, wait, ok)
	}

	// Navigation clears refs and loaded-html but NOT snapshot
	ts.SetRefMap(map[string]RefEntry{"e1": {Selector: "a", Role: "link", Name: "A"}})
	ts.OnMainFrameNavigated()
	if ts.RefCount() != 0 {
		t.Fatal("refs should be cleared on nav")
	}
	_, _, ok = ts.GetLoadedHtml()
	if ok {
		t.Fatal("loaded html should be cleared on nav")
	}
	if ts.GetLastSnapshot() != "old snapshot" {
		t.Fatal("snapshot should NOT be cleared on nav")
	}
}
