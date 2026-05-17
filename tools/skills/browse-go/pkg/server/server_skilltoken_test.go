package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"browse-go/pkg/browser"
	"browse-go/pkg/skilltoken"
)

func TestSkillTokenAuth(t *testing.T) {
	skilltoken.Reset()
	bm := browser.NewBrowserManager()
	s := New(bm, "root-token-123")

	// 1. Skill token should allow access with restricted scope
	tok := skilltoken.Mint("test-skill", "spawn-1", 60*time.Second, "read,navigate,interact,inspect,system")
	req := httptest.NewRequest("POST", "/command", strings.NewReader(`{"command":"status","args":[]}`))
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code == http.StatusUnauthorized {
		t.Fatal("skill token should be accepted")
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"ok":true`) {
		t.Fatalf("expected ok=true, got: %s", body)
	}

	// 2. Invalid skill token should 401
	req2 := httptest.NewRequest("POST", "/command", strings.NewReader(`{"command":"status","args":[]}`))
	req2.Header.Set("Authorization", "Bearer sk_invalidtoken123456789")
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid skill token, got %d", rr2.Code)
	}

	// 3. Root token should still work
	req3 := httptest.NewRequest("POST", "/command", strings.NewReader(`{"command":"status","args":[]}`))
	req3.Header.Set("Authorization", "Bearer root-token-123")
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("root token should work, got %d", rr3.Code)
	}
}

func TestSkillTokenScopeEnforcement(t *testing.T) {
	skilltoken.Reset()
	bm := browser.NewBrowserManager()
	s := New(bm, "root-token-123")

	// Skill token with only "read" scope should NOT be able to run "goto"
	tok := skilltoken.Mint("test-skill", "spawn-2", 60*time.Second, "read")
	req := httptest.NewRequest("POST", "/command", strings.NewReader(`{"command":"goto","args":["https://example.com"]}`))
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	body := rr.Body.String()
	if strings.Contains(body, `"ok":true`) {
		t.Fatal("read-only skill token should not be able to run goto")
	}
}
