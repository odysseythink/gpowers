package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"browse-go/pkg/browser"
)

func TestServerScopeRestriction(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	payload, _ := json.Marshal(map[string]interface{}{
		"command": "goto",
		"args":    []string{"https://example.com"},
	})

	// Request with only "read" scope — navigate should be denied
	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Browse-Scope", "read")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["ok"] != false {
		t.Fatalf("expected ok=false for scope-restricted command, got %v", body)
	}
	errStr, _ := body["error"].(string)
	if errStr == "" {
		t.Fatal("expected error message for scope denial")
	}
}

func TestServerScopeAllowed(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	// "status" requires "system" scope
	payload, _ := json.Marshal(map[string]interface{}{
		"command": "status",
		"args":    []string{},
	})

	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-Browse-Scope", "system")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("expected ok=true for allowed scope, got %v: %s", body, rr.Body.String())
	}
}

func TestServerScopeViaQueryParam(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	payload, _ := json.Marshal(map[string]interface{}{
		"command": "goto",
		"args":    []string{"https://example.com"},
	})

	// Pass scope via query param instead of header
	req := httptest.NewRequest(http.MethodPost, "/command?scope=read", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	var body map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &body)
	if body["ok"] != false {
		t.Fatal("expected scope restriction via query param to work")
	}
}

func TestServerRateLimit(t *testing.T) {
	os.Setenv("BROWSE_RATE_LIMIT_RPS", "1")
	os.Setenv("BROWSE_RATE_LIMIT_BURST", "1")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_RPS")
	defer os.Unsetenv("BROWSE_RATE_LIMIT_BURST")

	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	payload, _ := json.Marshal(map[string]interface{}{
		"command": "status",
		"args":    []string{},
	})

	// First request should succeed
	req1 := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(payload))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer test-token")
	rr1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr1, req1)

	var body1 map[string]interface{}
	json.Unmarshal(rr1.Body.Bytes(), &body1)
	if body1["ok"] != true {
		t.Fatalf("first request should succeed: %v", body1)
	}

	// Second request (same client, same command) should be rate limited
	req2 := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer test-token")
	rr2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr2, req2)

	var body2 map[string]interface{}
	json.Unmarshal(rr2.Body.Bytes(), &body2)
	if body2["ok"] != false {
		t.Fatalf("second request should be rate limited: %v", body2)
	}
	errStr, _ := body2["error"].(string)
	if errStr == "" || !bytes.Contains(rr2.Body.Bytes(), []byte("rate limited")) {
		t.Fatalf("expected rate limit error, got: %v", body2)
	}
}

func TestServerHealthIncludesRateLimit(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rl, ok := body["rateLimit"].(map[string]interface{})
	if !ok {
		t.Fatal("expected health to include rateLimit")
	}
	if _, ok := rl["enabled"]; !ok {
		t.Fatal("expected rateLimit.enabled")
	}
}

func TestServerAuditStats(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/audit-stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := body["total"]; !ok {
		t.Fatal("expected audit-stats to include total")
	}
}

func TestServerCORSScopeHeaders(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodOptions, "/command", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	allowedHeaders := rr.Header().Get("Access-Control-Allow-Headers")
	if !bytes.Contains([]byte(allowedHeaders), []byte("X-Browse-Scope")) {
		t.Fatalf("expected CORS to allow X-Browse-Scope header, got: %s", allowedHeaders)
	}
}
