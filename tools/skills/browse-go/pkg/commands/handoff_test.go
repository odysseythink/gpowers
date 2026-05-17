package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"browse-go/pkg/browser"
)

func TestHandoffNoActivePage(t *testing.T) {
	bm := browser.NewBrowserManager()
	r := NewRegistry()
	_, err := r.Execute(bm, "handoff", nil)
	if err == nil {
		t.Fatal("expected error for handoff with no active page")
	}
	if !strings.Contains(err.Error(), "no active page") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandoffMessage(t *testing.T) {
	if browser.FindChromiumExecutable() == "" {
		t.Skip("Chromium not found")
	}
	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Handoff Test</title></head><body><h1>Test</h1></body></html>`)
	}))
	defer ts.Close()

	r := NewRegistry()
	// Navigate to a page first
	_, err := r.Execute(bm, "goto", []string{ts.URL})
	if err != nil {
		t.Fatalf("goto failed: %v", err)
	}

	out, err := r.Execute(bm, "handoff", []string{"user taking over"})
	if err != nil {
		t.Fatalf("handoff failed: %v", err)
	}
	if !strings.Contains(out, "Handed off to user") {
		t.Fatalf("unexpected output: %q", out)
	}
	if !strings.Contains(out, "user taking over") {
		t.Fatalf("missing message in output: %q", out)
	}

	// Clean up headed instance
	_ = bm.CloseHeaded()
}

func TestResumeWithoutHandoff(t *testing.T) {
	if browser.FindChromiumExecutable() == "" {
		t.Skip("Chromium not found")
	}
	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	// Navigate so we have a session for snapshot
	r := NewRegistry()
	_, _ = r.Execute(bm, "goto", []string{"about:blank"})

	out, err := r.Execute(bm, "resume", nil)
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if !strings.Contains(out, "Resumed AI control") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestConnectDisconnect(t *testing.T) {
	if browser.FindChromiumExecutable() == "" {
		t.Skip("Chromium not found")
	}
	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	r := NewRegistry()
	out, err := r.Execute(bm, "connect", nil)
	if err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if !strings.Contains(out, "Connected to headed browser") {
		t.Fatalf("unexpected output: %q", out)
	}

	out, err = r.Execute(bm, "disconnect", nil)
	if err != nil {
		t.Fatalf("disconnect failed: %v", err)
	}
	if !strings.Contains(out, "Disconnected headed browser") {
		t.Fatalf("unexpected output: %q", out)
	}
}
