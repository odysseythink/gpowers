package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"

	"browse-go/pkg/cookieimport"
	"browse-go/pkg/picker"
)

// Picker domain tracking (mirrors TS importedDomains / importedCounts).
var (
	pickerMu          sync.Mutex
	pickerDomains     = make(map[string]bool)   // domain -> imported?
	pickerCounts      = make(map[string]int)    // domain -> count
)

// handleCookiePicker routes all /cookie-picker/* requests.
func (s *Server) handleCookiePicker(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/cookie-picker")
	if path == "" {
		path = "/"
	}

	// CORS preflight
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://127.0.0.1:%d", s.Port()))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// GET /cookie-picker — serve HTML (requires code or session cookie)
	if path == "/" && r.Method == http.MethodGet {
		s.handleCookiePickerPage(w, r)
		return
	}

	// ─── Auth gate: all data/action routes require Bearer token or session cookie ───
	if !s.checkPickerAuth(r) {
		pickerWriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	switch {
	case path == "/browsers" && r.Method == http.MethodGet:
		s.handlePickerBrowsers(w, r)
	case path == "/profiles" && r.Method == http.MethodGet:
		s.handlePickerProfiles(w, r)
	case path == "/domains" && r.Method == http.MethodGet:
		s.handlePickerDomains(w, r)
	case path == "/import" && r.Method == http.MethodPost:
		s.handlePickerImport(w, r)
	case path == "/remove" && r.Method == http.MethodPost:
		s.handlePickerRemove(w, r)
	case path == "/imported" && r.Method == http.MethodGet:
		s.handlePickerImported(w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// checkPickerAuth returns true if the request has a valid Bearer token or session cookie.
func (s *Server) checkPickerAuth(r *http.Request) bool {
	// Bearer token
	token := extractBearer(r)
	if token != "" && token == s.AuthToken() {
		return true
	}
	// Session cookie
	session := picker.ExtractSession(r)
	if session != "" && picker.ValidateSession(session) {
		return true
	}
	return false
}

// ─── Page ─────────────────────────────────────────────────────

func (s *Server) handleCookiePickerPage(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	// Code exchange: validate one-time code, set session cookie, redirect
	if code != "" {
		session, ok := picker.ConsumeCode(code)
		if !ok {
			http.Error(w, "Invalid or expired code. Re-run cookie-import-browser.", http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     picker.SessionCookieName(),
			Value:    session,
			Path:     "/cookie-picker",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   3600,
		})
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, "/cookie-picker", http.StatusFound)
		return
	}

	// Session cookie: serve HTML
	session := picker.ExtractSession(r)
	if session != "" && picker.ValidateSession(session) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(picker.GetHTML(s.Port())))
		return
	}

	// No code, no session: reject
	http.Error(w, "Access denied. Open the cookie picker from browse.", http.StatusForbidden)
}

// ─── Browsers ─────────────────────────────────────────────────

func (s *Server) handlePickerBrowsers(w http.ResponseWriter, r *http.Request) {
	browsers := cookieimport.FindInstalledBrowsers()
	var out []map[string]interface{}
	for _, b := range browsers {
		out = append(out, map[string]interface{}{
			"name":    b.Name,
			"aliases": b.Aliases,
		})
	}
	pickerWriteJSON(w, http.StatusOK, map[string]interface{}{"browsers": out})
}

// ─── Profiles ─────────────────────────────────────────────────

func (s *Server) handlePickerProfiles(w http.ResponseWriter, r *http.Request) {
	browserName := r.URL.Query().Get("browser")
	if browserName == "" {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing 'browser' parameter", "code": "missing_param"})
		return
	}
	profiles, err := cookieimport.ListProfiles(browserName)
	if err != nil {
		pickerWriteError(w, err)
		return
	}
	pickerWriteJSON(w, http.StatusOK, map[string]interface{}{"profiles": profiles})
}

// ─── Domains ──────────────────────────────────────────────────

func (s *Server) handlePickerDomains(w http.ResponseWriter, r *http.Request) {
	browserName := r.URL.Query().Get("browser")
	if browserName == "" {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing 'browser' parameter", "code": "missing_param"})
		return
	}
	profile := r.URL.Query().Get("profile")
	if profile == "" {
		profile = "Default"
	}
	result, err := cookieimport.ListDomains(browserName, profile)
	if err != nil {
		pickerWriteError(w, err)
		return
	}
	pickerWriteJSON(w, http.StatusOK, map[string]interface{}{
		"browser": result.Browser,
		"domains": result.Domains,
	})
}

// ─── Import ───────────────────────────────────────────────────

func (s *Server) handlePickerImport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Browser  string   `json:"browser"`
		Domains  []string `json:"domains"`
		Profile  string   `json:"profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body", "code": "bad_request"})
		return
	}
	if body.Browser == "" {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing 'browser' field", "code": "missing_param"})
		return
	}
	if len(body.Domains) == 0 {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing or empty 'domains' array", "code": "missing_param"})
		return
	}

	profile := body.Profile
	if profile == "" {
		profile = "Default"
	}

	// Decrypt cookies from browser DB
	result, err := cookieimport.ImportCookies(body.Browser, body.Domains, profile)
	if err != nil {
		pickerWriteError(w, err)
		return
	}

	// If all cookies failed and v20 encryption is detected, try CDP extraction
	if result.Count == 0 && result.Failed > 0 && cookieimport.HasV20Cookies(body.Browser, profile) {
		log.Printf("[cookie-picker] v20 App-Bound Encryption detected, trying CDP extraction...")
		var cdpErr error
		result, cdpErr = cookieimport.ImportCookiesViaCDP(body.Browser, body.Domains, profile)
		if cdpErr != nil {
			log.Printf("[cookie-picker] CDP fallback failed: %v", cdpErr)
			pickerWriteJSON(w, http.StatusOK, map[string]interface{}{
				"imported":     0,
				"failed":       result.Failed,
				"domainCounts": map[string]int{},
				"message":      fmt.Sprintf("Cookies use App-Bound Encryption (v20). Close %s, retry, or use /connect-chrome to browse with your real browser directly.", body.Browser),
				"code":         "v20_encryption",
			})
			return
		}
	}

	if result.Count == 0 {
		msg := "No cookies found for the specified domains"
		if result.Failed > 0 {
			msg = fmt.Sprintf("All %d cookies failed to decrypt", result.Failed)
		}
		pickerWriteJSON(w, http.StatusOK, map[string]interface{}{
			"imported":     0,
			"failed":       result.Failed,
			"domainCounts": map[string]int{},
			"message":      msg,
		})
		return
	}

	// Convert CDPCookie -> network.CookieParam and set in browser
	params := make([]*network.CookieParam, 0, len(result.Cookies))
	for _, c := range result.Cookies {
		cp := &network.CookieParam{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HTTPOnly,
			SameSite: c.SameSite,
		}
		if c.Expires >= 0 {
			t := cdp.TimeSinceEpoch(time.Unix(int64(c.Expires), 0))
			cp.Expires = &t
		}
		params = append(params, cp)
	}
	if err := s.bm.SetCDPCookies(params); err != nil {
		pickerWriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "code": "internal_error"})
		return
	}

	// Track what was imported
	pickerMu.Lock()
	for domain, count := range result.DomainCounts {
		pickerDomains[domain] = true
		pickerCounts[domain] += count
	}
	pickerMu.Unlock()
	s.bm.TrackCookieImportWithCounts(result.DomainCounts)

	log.Printf("[cookie-picker] Imported %d cookies for %d domains", result.Count, len(result.DomainCounts))

	pickerWriteJSON(w, http.StatusOK, map[string]interface{}{
		"imported":     result.Count,
		"failed":       result.Failed,
		"domainCounts": result.DomainCounts,
	})
}

// ─── Remove ───────────────────────────────────────────────────

func (s *Server) handlePickerRemove(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Domains []string `json:"domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON body", "code": "bad_request"})
		return
	}
	if len(body.Domains) == 0 {
		pickerWriteJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing or empty 'domains' array", "code": "missing_param"})
		return
	}

	for _, domain := range body.Domains {
		if err := s.bm.ClearCookiesForDomain(domain); err != nil {
			log.Printf("[cookie-picker] Failed to clear cookies for %s: %v", domain, err)
		}
	}

	pickerMu.Lock()
	for _, domain := range body.Domains {
		delete(pickerDomains, domain)
		delete(pickerCounts, domain)
	}
	pickerMu.Unlock()
	for _, domain := range body.Domains {
		s.bm.UntrackCookieImportDomain(domain)
	}

	log.Printf("[cookie-picker] Removed cookies for %d domains", len(body.Domains))

	pickerWriteJSON(w, http.StatusOK, map[string]interface{}{
		"removed":  len(body.Domains),
		"domains": body.Domains,
	})
}

// ─── Imported ─────────────────────────────────────────────────

func (s *Server) handlePickerImported(w http.ResponseWriter, r *http.Request) {
	pickerMu.Lock()
	entries := make([]map[string]interface{}, 0, len(pickerDomains))
	var totalCookies int
	for domain := range pickerDomains {
		count := pickerCounts[domain]
		entries = append(entries, map[string]interface{}{
			"domain": domain,
			"count":  count,
		})
		totalCookies += count
	}
	pickerMu.Unlock()

	// Sort by count descending (simple bubble for small N)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j]["count"].(int) > entries[i]["count"].(int) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	pickerWriteJSON(w, http.StatusOK, map[string]interface{}{
		"domains":      entries,
		"totalDomains": len(entries),
		"totalCookies": totalCookies,
	})
}

// ─── Helpers ──────────────────────────────────────────────────

func pickerWriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func pickerWriteError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	msg := err.Error()
	code := "internal_error"
	action := ""

	if cie, ok := err.(*cookieimport.Error); ok {
		code = cie.Code
		action = cie.Action
		if cie.Action == "retry" {
			status = http.StatusServiceUnavailable
		}
	}

	resp := map[string]interface{}{
		"error": msg,
		"code":  code,
	}
	if action != "" {
		resp["action"] = action
	}
	pickerWriteJSON(w, status, resp)
}
