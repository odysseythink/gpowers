package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"browse-go/pkg/browser"
)

func TestServerHealth(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["alive"] != true {
		t.Fatalf("expected alive=true, got %v", body["alive"])
	}
}

func TestServerAuthRequired(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	// /command without auth should 401
	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestServerCommandWithAuth(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	payload, _ := json.Marshal(map[string]interface{}{
		"command": "status",
		"args":    []string{},
	})
	req := httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	// Should succeed even without Chromium launched (status returns "unhealthy")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("expected ok=true, got %v: %s", body["ok"], rr.Body.String())
	}
}

func TestServerBatch(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	payload, _ := json.Marshal(map[string]interface{}{
		"commands": [][]string{
			{"status"},
			{"url"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/batch", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	results, ok := body["results"].([]interface{})
	if !ok || len(results) != 2 {
		t.Fatalf("expected 2 results, got %v", body["results"])
	}
}

func TestServerRefsAndTabs(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	for _, path := range []string{"/refs", "/tabs"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		rr := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, rr.Code)
		}
	}
}

func TestServerCORS(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS headers")
	}
}

func TestServerConnectAlive(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/connect", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]bool
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !body["alive"] {
		t.Fatal("expected alive=true")
	}
}

func TestServerPairRootOnly(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	// Without auth — middleware returns 401 before reaching handler
	req := httptest.NewRequest(http.MethodPost, "/pair", bytes.NewReader([]byte(`{}`)))
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	// With root auth
	payload, _ := json.Marshal(map[string]interface{}{"clientId": "test-agent"})
	req = httptest.NewRequest(http.MethodPost, "/pair", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["setup_key"] == nil || body["setup_key"] == "" {
		t.Fatal("expected setup_key")
	}
}

func TestServerTokenLifecycle(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	// Mint token
	payload, _ := json.Marshal(map[string]interface{}{
		"clientId": "alice",
		"scopes":   []string{"read", "write"},
	})
	req := httptest.NewRequest(http.MethodPost, "/token", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var mint map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &mint); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	token := mint["token"].(string)

	// List agents
	req = httptest.NewRequest(http.MethodGet, "/agents", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var list map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	agents := list["agents"].([]interface{})
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}

	// Use token for command
	cmdPayload, _ := json.Marshal(map[string]interface{}{"command": "status", "args": []string{}})
	req = httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(cmdPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Revoke via POST /token with action=revoke (since ServeMux doesn't do wildcards)
	revokePayload, _ := json.Marshal(map[string]interface{}{"clientId": "alice"})
	req = httptest.NewRequest(http.MethodPost, "/token", bytes.NewReader(revokePayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for revoke, got %d: %s", rr.Code, rr.Body.String())
	}

	// Token should no longer work
	req = httptest.NewRequest(http.MethodPost, "/command", bytes.NewReader(cmdPayload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after revoke, got %d", rr.Code)
	}
}

func TestServerBatchNestedGuard(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	payload, _ := json.Marshal(map[string]interface{}{
		"commands": [][]string{
			{"status"},
			{"batch"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/batch", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for nested batch, got %d", rr.Code)
	}
}


func TestServerWelcome(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/welcome", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html, got %s", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "GStack Browser") && !strings.Contains(body, "gstack") {
		t.Fatalf("unexpected welcome body: %s", body[:min(len(body), 200)])
	}
}

func TestServerSseSession(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodPost, "/sse-session", nil)
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
	if body["ok"] != true {
		t.Fatalf("expected ok=true, got %v", body["ok"])
	}
	// Check that Set-Cookie header is present
	cookies := rr.Header()["Set-Cookie"]
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header")
	}
	if !strings.Contains(cookies[0], "gstack_sse=") {
		t.Fatalf("expected gstack_sse cookie, got %s", cookies[0])
	}
}

func TestServerPtySessionWithoutAgent(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodPost, "/pty-session", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when agent not ready, got %d", rr.Code)
	}
}

func TestServerHealthHeadedMode(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "chrome-extension://test")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["token"] != "test-token" {
		t.Fatalf("expected token for extension origin, got %v", body["token"])
	}
	if body["chatEnabled"] != false {
		t.Fatalf("expected chatEnabled=false, got %v", body["chatEnabled"])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
