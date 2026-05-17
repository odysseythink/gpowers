package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"browse-go/pkg/browser"
	"browse-go/pkg/commands"
	"browse-go/pkg/server"
)

// hasChromium returns true if a Chromium executable is available.
func hasChromium() bool {
	return browser.FindChromiumExecutable() != ""
}

// testPageServer returns a local HTTP server serving a simple HTML page.
func testPageServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><title>TestPage</title></head>
<body>
<h1>Hello E2E</h1>
<p>paragraph one</p>
<a href="/link">link text</a>
<button id="btn">Click me</button>
<input type="text" placeholder="type here" />
</body></html>`)
	}))
}

// TestE2ELaunchAndNavigate tests the full lifecycle: launch → navigate → extract.
func TestE2ELaunchAndNavigate(t *testing.T) {
	if !hasChromium() {
		t.Skip("Chromium not found, skipping e2e test")
	}

	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	ts := testPageServer()
	defer ts.Close()

	registry := commands.NewRegistry()
	_, err := registry.Execute(bm, "goto", []string{ts.URL})
	if err != nil {
		t.Fatalf("goto failed: %v", err)
	}

	// Wait for load
	_, err = registry.Execute(bm, "wait", []string{"--load"})
	if err != nil {
		t.Fatalf("wait failed: %v", err)
	}

	// Verify URL
	result, err := registry.Execute(bm, "url", nil)
	if err != nil {
		t.Fatalf("url failed: %v", err)
	}
	if !strings.Contains(result, ts.URL) {
		t.Fatalf("expected URL containing %s, got: %s", ts.URL, result)
	}

	// Extract text
	result, err = registry.Execute(bm, "text", nil)
	if err != nil {
		t.Fatalf("text failed: %v", err)
	}
	if !strings.Contains(result, "Hello E2E") {
		t.Fatalf("expected 'Hello E2E' in text, got: %s", result)
	}

	// Extract links
	result, err = registry.Execute(bm, "links", nil)
	if err != nil {
		t.Fatalf("links failed: %v", err)
	}
	if !strings.Contains(result, "link text") {
		t.Fatalf("expected 'link text' in links, got: %s", result)
	}
}

// TestE2EScreenshot tests screenshot generation.
func TestE2EScreenshot(t *testing.T) {
	if !hasChromium() {
		t.Skip("Chromium not found, skipping e2e test")
	}

	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	ts := testPageServer()
	defer ts.Close()

	registry := commands.NewRegistry()
	_, _ = registry.Execute(bm, "goto", []string{ts.URL})
	_, _ = registry.Execute(bm, "wait", []string{"--load"})

	// Screenshot to file
	screenshotPath := fmt.Sprintf("%s/browse-e2e-test-%d.png", os.TempDir(), time.Now().UnixNano())
	result, err := registry.Execute(bm, "screenshot", []string{screenshotPath})
	if err != nil {
		t.Fatalf("screenshot failed: %v", err)
	}
	if !strings.Contains(result, "saved") && !strings.Contains(result, screenshotPath) {
		t.Fatalf("unexpected screenshot result: %s", result)
	}
	defer os.Remove(screenshotPath)

	// Verify file was created and has PNG magic bytes
	data, err := os.ReadFile(screenshotPath)
	if err != nil {
		t.Fatalf("screenshot file not found: %v", err)
	}
	if len(data) < 8 {
		t.Fatalf("screenshot too small (%d bytes)", len(data))
	}
	// PNG starts with 0x89 PNG, JPEG with 0xFF 0xD8
	isPNG := data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47
	isJPEG := data[0] == 0xFF && data[1] == 0xD8
	if !isPNG && !isJPEG {
		t.Fatalf("screenshot is not a valid PNG or JPEG (first bytes: %x)", data[:8])
	}
}

// TestE2ESnapshot tests accessibility snapshot generation.
func TestE2ESnapshot(t *testing.T) {
	if !hasChromium() {
		t.Skip("Chromium not found, skipping e2e test")
	}

	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	ts := testPageServer()
	defer ts.Close()

	registry := commands.NewRegistry()
	_, _ = registry.Execute(bm, "goto", []string{ts.URL})
	_, _ = registry.Execute(bm, "wait", []string{"--load"})

	result, err := registry.Execute(bm, "snapshot", []string{"-i"})
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if !strings.Contains(result, "@e1") {
		t.Fatalf("expected @e1 ref in snapshot, got: %s", result)
	}

	// Test ref resolution via click
	_, err = registry.Execute(bm, "click", []string{"@e1"})
	if err != nil {
		t.Fatalf("click @e1 failed: %v", err)
	}
}

// TestE2ETabs tests tab management.
func TestE2ETabs(t *testing.T) {
	if !hasChromium() {
		t.Skip("Chromium not found, skipping e2e test")
	}

	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	ts := testPageServer()
	defer ts.Close()

	registry := commands.NewRegistry()

	// Start with one tab
	result, err := registry.Execute(bm, "tabs", nil)
	if err != nil {
		t.Fatalf("tabs failed: %v", err)
	}
	if !strings.Contains(result, "[1]") {
		t.Fatalf("expected [1] in tabs output, got: %s", result)
	}

	// Create a new tab
	_, err = registry.Execute(bm, "newtab", []string{ts.URL + "/tab2"})
	if err != nil {
		t.Fatalf("newtab failed: %v", err)
	}

	result, err = registry.Execute(bm, "tabs", nil)
	if err != nil {
		t.Fatalf("tabs after newtab failed: %v", err)
	}
	if !strings.Contains(result, "[2]") {
		t.Fatalf("expected [2] in tabs output, got: %s", result)
	}

	// Switch to tab 1
	_, err = registry.Execute(bm, "tab", []string{"1"})
	if err != nil {
		t.Fatalf("tab switch failed: %v", err)
	}

	// Close tab 2
	_, err = registry.Execute(bm, "closetab", []string{"2"})
	if err != nil {
		t.Fatalf("closetab failed: %v", err)
	}
}

// TestE2EServerEndpoints tests the HTTP API with a real browser.
func TestE2EServerEndpoints(t *testing.T) {
	if !hasChromium() {
		t.Skip("Chromium not found, skipping e2e test")
	}

	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	// Navigate somewhere first
	registry := commands.NewRegistry()
	_, _ = registry.Execute(bm, "goto", []string{"about:blank"})

	srv := server.New(bm, "e2e-test-token")

	// Health (no auth required)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("health: expected 200, got %d", rr.Code)
	}

	// Command with auth
	payload, _ := json.Marshal(map[string]interface{}{
		"command": "status",
		"args":    []string{},
	})
	req = httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer e2e-test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("command: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("command: invalid JSON: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("command: expected ok=true, got %v", body["ok"])
	}

	// Tabs endpoint
	req = httptest.NewRequest(http.MethodGet, "/tabs", nil)
	req.Header.Set("Authorization", "Bearer e2e-test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("tabs: expected 200, got %d", rr.Code)
	}

	// Refs endpoint
	req = httptest.NewRequest(http.MethodGet, "/refs", nil)
	req.Header.Set("Authorization", "Bearer e2e-test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("refs: expected 200, got %d", rr.Code)
	}
}

// TestE2EInboxCommand tests the inbox command.
func TestE2EInboxCommand(t *testing.T) {
	bm := browser.NewBrowserManager() // no launch needed
	registry := commands.NewRegistry()

	result, err := registry.Execute(bm, "inbox", nil)
	if err != nil {
		t.Fatalf("inbox failed: %v", err)
	}
	// Should either say "Inbox empty." or "Not in a git repository"
	if !strings.Contains(result, "Inbox empty") && !strings.Contains(result, "Not in a git repository") {
		t.Fatalf("unexpected inbox result: %s", result)
	}
}

// TestE2ECdpCommand tests the CDP command allowlist.
func TestE2ECdpCommand(t *testing.T) {
	if !hasChromium() {
		t.Skip("Chromium not found, skipping e2e test")
	}

	bm := browser.NewBrowserManager()
	if err := bm.Launch(); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	defer bm.Close()

	ts := testPageServer()
	defer ts.Close()

	registry := commands.NewRegistry()
	_, _ = registry.Execute(bm, "goto", []string{ts.URL})
	_, _ = registry.Execute(bm, "wait", []string{"--load"})

	// Enable Performance domain first
	_, err := registry.Execute(bm, "cdp", []string{"Performance.enable", "{}"})
	if err != nil {
		t.Fatalf("cdp Performance.enable failed: %v", err)
	}

	// Allowed method: Performance.getMetrics
	result, err := registry.Execute(bm, "cdp", []string{"Performance.getMetrics", "{}"})
	if err != nil {
		t.Fatalf("cdp Performance.getMetrics failed: %v", err)
	}
	if !strings.Contains(result, "Timestamp") && !strings.Contains(result, "JSHeapUsedSize") {
		t.Fatalf("expected performance metrics in result, got: %s", result)
	}

	// Denied method should fail
	_, err = registry.Execute(bm, "cdp", []string{"Runtime.evaluate", `{"expression":"1"}`})
	if err == nil {
		t.Fatal("expected error for denied Runtime.evaluate")
	}
	if !strings.Contains(err.Error(), "DENIED") {
		t.Fatalf("expected DENIED error, got: %v", err)
	}

	// Help should work
	result, err = registry.Execute(bm, "cdp", []string{"help"})
	if err != nil {
		t.Fatalf("cdp help failed: %v", err)
	}
	if !strings.Contains(result, "Usage") {
		t.Fatalf("expected help text, got: %s", result)
	}
}

// TestE2EFileEndpoint tests the GET /file endpoint.
func TestE2EFileEndpoint(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := server.New(bm, "file-test-token")

	// Create a temp file — resolve through symlinks to match server logic
	tmpFile := fmt.Sprintf("%s/browse-e2e-file-%d.txt", os.TempDir(), time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, []byte("hello browse file"), 0644); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	defer os.Remove(tmpFile)

	// Resolve the path the same way the server does
	resolvedFile, _ := filepath.EvalSymlinks(tmpFile)
	if resolvedFile == "" {
		resolvedFile = tmpFile
	}

	// Valid file request
	req := httptest.NewRequest(http.MethodGet, "/file?path="+resolvedFile, nil)
	req.Header.Set("Authorization", "Bearer file-test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("file: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "hello browse file" {
		t.Fatalf("unexpected file content: %s", rr.Body.String())
	}

	// Missing path
	req = httptest.NewRequest(http.MethodGet, "/file", nil)
	req.Header.Set("Authorization", "Bearer file-test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("file missing path: expected 400, got %d", rr.Code)
	}

	// Path outside temp dir
	req = httptest.NewRequest(http.MethodGet, "/file?path=/etc/passwd", nil)
	req.Header.Set("Authorization", "Bearer file-test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("file escape: expected 403, got %d", rr.Code)
	}

	// Nonexistent file
	req = httptest.NewRequest(http.MethodGet, "/file?path="+os.TempDir()+"/browse-nonexistent-12345.txt", nil)
	req.Header.Set("Authorization", "Bearer file-test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("file not found: expected 404, got %d", rr.Code)
	}
}
