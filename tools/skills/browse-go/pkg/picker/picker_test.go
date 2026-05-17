package picker

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateAndConsumeCode(t *testing.T) {
	code := GenerateCode()
	if code == "" {
		t.Fatal("expected non-empty code")
	}
	if len(code) < 32 {
		t.Fatalf("expected code length >= 32, got %d", len(code))
	}

	// Consume should succeed
	session, ok := ConsumeCode(code)
	if !ok {
		t.Fatal("expected code to be valid")
	}
	if session == "" {
		t.Fatal("expected non-empty session")
	}

	// Second consume should fail (one-time)
	_, ok = ConsumeCode(code)
	if ok {
		t.Fatal("expected code to be consumed already")
	}
}

func TestConsumeInvalidCode(t *testing.T) {
	_, ok := ConsumeCode("not-a-valid-code")
	if ok {
		t.Fatal("expected invalid code to fail")
	}
}

func TestCodeExpiry(t *testing.T) {
	code := GenerateCode()

	// Expire the code immediately
	codeMu.Lock()
	pendingCodes[code] = time.Now().UnixMilli() - 1
	codeMu.Unlock()

	_, ok := ConsumeCode(code)
	if ok {
		t.Fatal("expected expired code to fail")
	}
}

func TestSessionValidation(t *testing.T) {
	code := GenerateCode()
	session, ok := ConsumeCode(code)
	if !ok {
		t.Fatal("setup failed")
	}

	if !ValidateSession(session) {
		t.Fatal("expected fresh session to be valid")
	}

	// Expire the session
	sessionMu.Lock()
	validSessions[session] = time.Now().UnixMilli() - 1
	sessionMu.Unlock()

	if ValidateSession(session) {
		t.Fatal("expected expired session to be invalid")
	}
}

func TestHasActive(t *testing.T) {
	// Clear all state
	codeMu.Lock()
	for k := range pendingCodes {
		delete(pendingCodes, k)
	}
	codeMu.Unlock()
	sessionMu.Lock()
	for k := range validSessions {
		delete(validSessions, k)
	}
	sessionMu.Unlock()

	if HasActive() {
		t.Fatal("expected no active picker initially")
	}

	code := GenerateCode()
	if !HasActive() {
		t.Fatal("expected active after generating code")
	}

	ConsumeCode(code)
	if !HasActive() {
		t.Fatal("expected active after consuming code (session created)")
	}
}

func TestExtractSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if ExtractSession(req) != "" {
		t.Fatal("expected no session without cookie")
	}

	req.AddCookie(&http.Cookie{Name: SessionCookieName(), Value: "test-session"})
	if ExtractSession(req) != "test-session" {
		t.Fatal("expected session from cookie")
	}
}

func TestGetHTML(t *testing.T) {
	html := GetHTML(12345)
	if !strings.Contains(html, "localhost:12345") {
		t.Fatal("expected HTML to contain port")
	}
	if !strings.Contains(html, "http://127.0.0.1:12345") {
		t.Fatal("expected HTML to contain base URL")
	}
	if !strings.Contains(html, "Cookie Import") {
		t.Fatal("expected HTML to contain title")
	}
	if !strings.Contains(html, "Source Browser") {
		t.Fatal("expected HTML to contain left panel header")
	}
	if !strings.Contains(html, "Imported to Session") {
		t.Fatal("expected HTML to contain right panel header")
	}
}
