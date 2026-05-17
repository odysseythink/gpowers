package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"browse-go/pkg/browser"
	"browse-go/pkg/picker"
)

func TestCookiePickerPageNoAuth(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/cookie-picker", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without code/session, got %d", rr.Code)
	}
}

func TestCookiePickerPageWithCode(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	code := picker.GenerateCode()
	req := httptest.NewRequest(http.MethodGet, "/cookie-picker?code="+code, nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	// Should redirect to /cookie-picker (setting session cookie)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if loc != "/cookie-picker" {
		t.Fatalf("expected redirect to /cookie-picker, got %s", loc)
	}

	// Extract session cookie
	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == picker.SessionCookieName() {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("expected HttpOnly session cookie")
	}

	// Now use the session cookie to access the page
	req2 := httptest.NewRequest(http.MethodGet, "/cookie-picker", nil)
	req2.AddCookie(sessionCookie)
	rr2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 with session, got %d: %s", rr2.Code, rr2.Body.String())
	}
	if !strings.Contains(rr2.Body.String(), "Cookie Import") {
		t.Fatal("expected HTML response")
	}
}

func TestCookiePickerCodeReuse(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	code := picker.GenerateCode()

	// First use
	req1 := httptest.NewRequest(http.MethodGet, "/cookie-picker?code="+code, nil)
	rr1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusFound {
		t.Fatalf("expected 302 on first use, got %d", rr1.Code)
	}

	// Second use should fail
	req2 := httptest.NewRequest(http.MethodGet, "/cookie-picker?code="+code, nil)
	rr2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on reuse, got %d", rr2.Code)
	}
}

func TestCookiePickerBrowsers(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/cookie-picker/browsers", nil)
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
	browsers, ok := body["browsers"].([]interface{})
	if !ok {
		t.Fatalf("expected browsers array, got %T", body["browsers"])
	}
	// May be empty on this machine, but structure should be valid
	_ = browsers
}

func TestCookiePickerProfilesMissingParam(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/cookie-picker/profiles", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCookiePickerDomainsMissingParam(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/cookie-picker/domains", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCookiePickerImportBadRequest(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	// Missing body
	req := httptest.NewRequest(http.MethodPost, "/cookie-picker/import", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCookiePickerRemoveBadRequest(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodPost, "/cookie-picker/remove", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestCookiePickerImportedEmpty(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodGet, "/cookie-picker/imported", nil)
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
	if body["totalDomains"].(float64) != 0 {
		t.Fatalf("expected 0 domains, got %v", body["totalDomains"])
	}
}

func TestCookiePickerSessionAuth(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	// Create a session via code exchange
	code := picker.GenerateCode()
	req1 := httptest.NewRequest(http.MethodGet, "/cookie-picker?code="+code, nil)
	rr1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr1, req1)

	var sessionCookie *http.Cookie
	for _, c := range rr1.Result().Cookies() {
		if c.Name == picker.SessionCookieName() {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("setup: expected session cookie")
	}

	// Use session cookie for /browsers (no Bearer token)
	req2 := httptest.NewRequest(http.MethodGet, "/cookie-picker/browsers", nil)
	req2.AddCookie(sessionCookie)
	rr2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200 with session cookie, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestCookiePickerCORS(t *testing.T) {
	bm := browser.NewBrowserManager()
	srv := New(bm, "test-token")

	req := httptest.NewRequest(http.MethodOptions, "/cookie-picker/browsers", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS headers")
	}
}
