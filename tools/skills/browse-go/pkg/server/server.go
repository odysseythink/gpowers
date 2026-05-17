// Package server provides the HTTP API for the browse daemon.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"browse-go/pkg/activity"
	"browse-go/pkg/browser"
	"browse-go/pkg/commands"
	"browse-go/pkg/config"
	"browse-go/pkg/inspector"
	"browse-go/pkg/security"
	"browse-go/pkg/skilltoken"
	"browse-go/pkg/terminal"
	"browse-go/pkg/tokenregistry"
	"browse-go/pkg/tunnel"
)

// Server is the browse HTTP daemon.
type Server struct {
	mu            sync.RWMutex
	bm            *browser.BrowserManager
	registry      *commands.Registry
	tokenRegistry *tokenregistry.Registry
	inspector     *inspector.Inspector
	authToken     string
	port          int
	listener      net.Listener
	httpServer    *http.Server
	idleTimer     *time.Timer
	idleTimeout   time.Duration
	startTime     time.Time

	// Tunnel surface (optional)
	tunnel       *tunnel.Tunnel
	tunnelActive bool
	tunnelURL    string
}

// New creates a new Server. If authToken is empty, a random 32-byte token is generated.
func New(bm *browser.BrowserManager, authToken string) *Server {
	if authToken == "" {
		authToken = generateToken()
	}
	tr := tokenregistry.New()
	_ = tr.Init(authToken) // root token = auth token
	return &Server{
		bm:            bm,
		registry:      commands.NewRegistry(),
		tokenRegistry: tr,
		inspector:     inspector.New(),
		authToken:     authToken,
		idleTimeout:   30 * time.Minute,
		startTime:     time.Now(),
	}
}

// SetIdleTimeout configures the auto-shutdown timer (default 30m).
func (s *Server) SetIdleTimeout(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.idleTimeout = d
}

// Port returns the bound listen port.
func (s *Server) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.port
}

// AuthToken returns the bearer token clients must present.
func (s *Server) AuthToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authToken
}

// Handler returns the root HTTP handler for testing or external serving.
func (s *Server) Handler() http.Handler {
	return s.mux()
}

// Start binds to a TCP port and begins serving HTTP.
// If port is 0, a random port between 10000-60000 is chosen.
func (s *Server) Start(port int) error {
	if port == 0 {
		port = pickPort()
	}

	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}

	s.mu.Lock()
	s.port = ln.Addr().(*net.TCPAddr).Port
	s.listener = ln
	s.httpServer = &http.Server{
		Handler:      s.withMiddleware(s.mux()),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	s.mu.Unlock()
	s.resetIdleTimer()

	log.Printf("[browse] Server listening on http://127.0.0.1:%d", s.Port())
	if s.tunnelActive && s.tunnelURL != "" {
		log.Printf("[browse] Tunnel active: %s", s.tunnelURL)
	}
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[browse] Server error: %v", err)
		}
	}()

	// Watch-mode snapshot ticker
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if !s.bm.IsWatching() {
				continue
			}
			result, err := s.registry.Execute(s.bm, "snapshot", []string{"-i"})
			if err != nil {
				continue
			}
			s.bm.AddWatchSnapshot(result)
		}
	}()

	// Tab-awareness state file ticker
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		cfg := config.Resolve(nil)
		for range ticker.C {
			s.bm.WriteTabState(cfg.StateDir)
		}
	}()

	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	srv := s.httpServer
	s.mu.Unlock()
	if srv != nil {
		return srv.Shutdown(ctx)
	}
	return nil
}

// mux builds the HTTP route table.
func (s *Server) mux() *http.ServeMux {
	root := http.NewServeMux()

	// Cookie picker routes (own auth system, isolated from /command tokens)
	root.HandleFunc("/cookie-picker", s.handleCookiePickerWithIdle)
	root.HandleFunc("/cookie-picker/", s.handleCookiePickerWithIdle)

	// Unauthenticated routes
	root.HandleFunc("/connect", s.handleConnect)
	root.HandleFunc("/welcome", s.handleWelcome)

	// API routes (standard Bearer auth + middleware)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/health", s.handleHealth)
	apiMux.HandleFunc("/command", s.handleCommand)
	apiMux.HandleFunc("/batch", s.handleBatch)
	apiMux.HandleFunc("/refs", s.handleRefs)
	apiMux.HandleFunc("/tabs", s.handleTabs)
	apiMux.HandleFunc("/file", s.handleFile)
	apiMux.HandleFunc("/stop", s.handleStop)
	apiMux.HandleFunc("/audit", s.handleAudit)
	apiMux.HandleFunc("/audit-stats", s.handleAuditStats)
	apiMux.HandleFunc("/activity/stream", activity.HandleStream)
	apiMux.HandleFunc("/activity/history", activity.HandleHistory)

	// Token registry routes (root-only)
	apiMux.HandleFunc("/pair", s.handlePair)
	apiMux.HandleFunc("/token", s.handleToken)
	apiMux.HandleFunc("/agents", s.handleAgents)

	// Inspector routes
	apiMux.HandleFunc("/inspector/pick", s.handleInspectorPick)
	apiMux.HandleFunc("/inspector/apply", s.handleInspectorApply)
	apiMux.HandleFunc("/inspector/reset", s.handleInspectorReset)
	apiMux.HandleFunc("/inspector/history", s.handleInspectorHistory)
	apiMux.HandleFunc("/inspector/events", s.handleInspectorEvents)

	// Session routes
	apiMux.HandleFunc("/sse-session", s.handleSseSession)
	apiMux.HandleFunc("/pty-session", s.handlePtySession)

	// Tunnel management
	apiMux.HandleFunc("/tunnel/start", s.handleTunnelStart)

	root.Handle("/", s.withMiddleware(apiMux))

	return root
}

// handleCookiePickerWithIdle wraps the picker handler with idle timer reset.
func (s *Server) handleCookiePickerWithIdle(w http.ResponseWriter, r *http.Request) {
	s.resetIdleTimer()
	s.handleCookiePicker(w, r)
}

// withMiddleware wraps the handler with CORS, auth, and activity tracking.
func (s *Server) withMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.resetIdleTimer()

		// CORS — allow local Chrome extension
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Browse-Scope, X-Client-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Auth (except /health GET)
		if !(r.URL.Path == "/health" && r.Method == http.MethodGet) {
			token := extractBearer(r)
			// Fallback to SSE session cookie for EventSource requests
			if token == "" {
				if c, err := r.Cookie("gstack_sse"); err == nil && c.Value != "" {
					token = c.Value
				}
			}
			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			// Root token
			if s.tokenRegistry.IsRoot(token) {
				// ok — root has all scopes
				r.Header.Set("X-Browse-Scope", "all")
				r.Header.Set("X-Client-ID", "root")
			} else if skilltoken.IsSkillToken(token) {
				// Skill-scoped token
				clientID, scopes, ok := skilltoken.ValidateString(token)
				if !ok {
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "invalid skill token"})
					return
				}
				r.Header.Set("X-Browse-Scope", scopes)
				r.Header.Set("X-Client-ID", clientID)
			} else if info := s.tokenRegistry.Validate(token); info != nil {
				// Registry-scoped token (pair-agent / session token)
				scopeStr := registryScopesToString(info.Scopes)
				r.Header.Set("X-Browse-Scope", scopeStr)
				r.Header.Set("X-Client-ID", info.ClientID)
			} else {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}

// ─── Handlers ─────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status := security.GetStatus()
	rateLimit := s.registry.RateLimitStatus()

	origin := r.Header.Get("Origin")
	isHeaded := s.bm.GetConnectionMode() == "headed"
	isExtension := strings.HasPrefix(origin, "chrome-extension://")

	resp := map[string]interface{}{
		"alive":     true,
		"healthy":   s.bm.IsHealthy(),
		"mode":      s.bm.GetConnectionMode(),
		"url":       s.bm.CurrentURL(),
		"tabs":      s.bm.TabCount(),
		"port":      s.Port(),
		"security":  status,
		"rateLimit": rateLimit,
		"chatEnabled": false,
		"terminalPort": s.readTerminalPort(),
	}
	if isHeaded || isExtension {
		resp["token"] = s.AuthToken()
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	opts := s.extractExecuteOpts(r)

	// Registry token: additional domain/rate checks
	token := extractBearer(r)
	if info := s.tokenRegistry.Validate(token); info != nil && info.ClientID != "root" {
		// Scope check via registry (chain subcommands checked at dispatch)
		if !s.tokenRegistry.CheckScope(info, req.Command) {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"ok": false, "error": fmt.Sprintf("command %q not allowed by token scope", req.Command),
			})
			return
		}
		// Rate limit check
		if allowed, retryAfter := s.tokenRegistry.CheckRate(info); !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"ok": false, "error": "rate limited",
			})
			return
		}
		// Domain check (if command involves a URL)
		pageURL := s.bm.CurrentURL()
		if pageURL != "" && !s.tokenRegistry.CheckDomain(info, pageURL) {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"ok": false, "error": "page domain not allowed by token restrictions",
			})
			return
		}
		// Record command
		defer s.tokenRegistry.RecordCommand(token)
	}

	result, err := s.registry.ExecuteWithOpts(s.bm, req.Command, req.Args, opts)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "result": result})
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Commands [][]string `json:"commands"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Pre-validation: reject nested batch
	for _, cmd := range req.Commands {
		if len(cmd) > 0 && cmd[0] == "batch" {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{
				"ok": false, "error": "nested batch commands are not allowed",
			})
			return
		}
	}

	opts := s.extractExecuteOpts(r)

	// Registry token validation (do once for the whole batch)
	token := extractBearer(r)
	var regInfo *tokenregistry.TokenInfo
	if info := s.tokenRegistry.Validate(token); info != nil && info.ClientID != "root" {
		regInfo = info
		// Rate limit check (one check for the batch)
		if allowed, retryAfter := s.tokenRegistry.CheckRate(info); !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"ok": false, "error": "rate limited",
			})
			return
		}
		// Domain check
		pageURL := s.bm.CurrentURL()
		if pageURL != "" && !s.tokenRegistry.CheckDomain(info, pageURL) {
			writeJSON(w, http.StatusForbidden, map[string]interface{}{
				"ok": false, "error": "page domain not allowed by token restrictions",
			})
			return
		}
	}

	results := make([]map[string]interface{}, len(req.Commands))
	for i, cmd := range req.Commands {
		if len(cmd) == 0 {
			results[i] = map[string]interface{}{"ok": false, "error": "empty command"}
			continue
		}
		name := cmd[0]
		var args []string
		if len(cmd) > 1 {
			args = cmd[1:]
		}

		// Per-command registry scope check
		if regInfo != nil && !s.tokenRegistry.CheckScope(regInfo, name) {
			results[i] = map[string]interface{}{
				"ok": false, "error": fmt.Sprintf("command %q not allowed by token scope", name),
			}
			continue
		}

		result, err := s.registry.ExecuteWithOpts(s.bm, name, args, opts)
		if err != nil {
			results[i] = map[string]interface{}{"ok": false, "error": err.Error()}
		} else {
			results[i] = map[string]interface{}{"ok": true, "result": result}
		}
	}

	if regInfo != nil {
		s.tokenRegistry.RecordCommand(token)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func (s *Server) handleRefs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	refs := s.bm.RefEntries()
	writeJSON(w, http.StatusOK, map[string]interface{}{"refs": refs})
}

func (s *Server) handleTabs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	list := s.bm.TabList()
	writeJSON(w, http.StatusOK, map[string]interface{}{"tabs": list})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "result": "Shutting down..."})
	go func() {
		time.Sleep(100 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
		_ = s.bm.Close()
	}()
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	opts := security.QueryAuditOptions{}
	if v := r.URL.Query().Get("command"); v != "" {
		opts.Command = v
	}
	if v := r.URL.Query().Get("client"); v != "" {
		opts.ClientID = v
	}
	if v := r.URL.Query().Get("verdict"); v != "" {
		opts.Verdict = v
	}
	if v := r.URL.Query().Get("max"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			opts.MaxResults = n
		}
	}

	records, err := security.QueryAudit(opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"records": records})
}

func (s *Server) handleAuditStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stats := security.AuditStats()
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing 'path' query parameter"})
		return
	}

	// Resolve and validate path is within temp dir
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Invalid path"})
		return
	}
	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "File not found"})
			return
		}
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot resolve path"})
		return
	}
	tempDir := os.TempDir()
	resolvedTempDir, _ := filepath.EvalSymlinks(tempDir)
	if resolvedTempDir == "" {
		resolvedTempDir = tempDir
	}
	if !strings.HasPrefix(realPath, tempDir) && !strings.HasPrefix(realPath, resolvedTempDir) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Path escapes temp directory"})
		return
	}

	stat, err := os.Stat(realPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "File not found"})
		} else {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		}
		return
	}
	if stat.IsDir() {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Path is a directory"})
		return
	}
	if stat.Size() > 200*1024*1024 {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "File too large (max 200MB)"})
		return
	}

	ext := strings.ToLower(filepath.Ext(realPath))
	mimeMap := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".webp": "image/webp", ".svg": "image/svg+xml",
		".avif": "image/avif",
		".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
		".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg",
		".pdf": "application/pdf", ".json": "application/json",
		".html": "text/html", ".txt": "text/plain", ".mhtml": "message/rfc822",
	}
	contentType := mimeMap[ext]
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(realPath)))
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	data, err := os.ReadFile(realPath)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
}

// ─── Helpers ──────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	// Also accept token in query param for SSE / EventSource
	return r.URL.Query().Get("token")
}

// extractExecuteOpts extracts scope and client ID from the HTTP request.
func (s *Server) extractExecuteOpts(r *http.Request) *commands.ExecuteOpts {
	scopeStr := r.Header.Get("X-Browse-Scope")
	if scopeStr == "" {
		scopeStr = r.URL.Query().Get("scope")
	}
	clientID := r.Header.Get("X-Client-ID")
	if clientID == "" {
		clientID = r.URL.Query().Get("client_id")
	}
	return &commands.ExecuteOpts{
		ScopeSet: security.NewScopeSet(scopeStr),
		ClientID: clientID,
		Port:     s.Port(),
	}
}

func (s *Server) resetIdleTimer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	if s.idleTimeout <= 0 {
		return
	}
	s.idleTimer = time.AfterFunc(s.idleTimeout, func() {
		log.Println("[browse] Idle timeout reached, shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
		_ = s.bm.Close()
		os.Exit(0)
	})
}

func pickPort() int {
	// Try random port in 10000-60000 range
	for i := 0; i < 20; i++ {
		port := 10000 + (os.Getpid()+i*997)%50000
		ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

// ─── Token Registry Handlers ──────────────────────────────

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	s.resetIdleTimer()

	if r.Method == http.MethodGet {
		if !s.tokenRegistry.CheckConnectRateLimit() {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Rate limited"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"alive": true})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.tokenRegistry.CheckConnectRateLimit() {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{
			"error": "Too many connection attempts. Wait 1 minute.",
		})
		return
	}

	var req struct {
		SetupKey string `json:"setup_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}
	if req.SetupKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing setup_key"})
		return
	}

	session, err := s.tokenRegistry.ExchangeSetupKey(req.SetupKey, nil)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "Invalid, expired, or already-used setup key",
		})
		return
	}

	log.Printf("[browse] Remote agent connected: %s (scopes: %s)", session.ClientID, registryScopesToString(session.Scopes))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":  session.Token,
		"expires": formatTimePtr(session.ExpiresAt),
		"scopes": session.Scopes,
		"agent":  session.ClientID,
	})
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.isRootRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Root token required"})
		return
	}

	var req struct {
		ClientID string                `json:"clientId"`
		Control  bool                  `json:"control,omitempty"`
		Admin    bool                  `json:"admin,omitempty"`
		Scopes   []tokenregistry.ScopeCategory `json:"scopes,omitempty"`
		Domains  []string              `json:"domains,omitempty"`
		RateLimit int                  `json:"rateLimit,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	var scopes []tokenregistry.ScopeCategory
	if len(req.Scopes) > 0 {
		scopes = req.Scopes
	} else if req.Control || req.Admin {
		scopes = []tokenregistry.ScopeCategory{
			tokenregistry.ScopeRead, tokenregistry.ScopeWrite,
			tokenregistry.ScopeAdmin, tokenregistry.ScopeMeta, tokenregistry.ScopeControl,
		}
	} else {
		scopes = []tokenregistry.ScopeCategory{
			tokenregistry.ScopeRead, tokenregistry.ScopeWrite,
			tokenregistry.ScopeAdmin, tokenregistry.ScopeMeta,
		}
	}

	setupKey, err := s.tokenRegistry.CreateSetupKey(tokenregistry.CreateTokenOptions{
		ClientID:  req.ClientID,
		Scopes:    scopes,
		Domains:   req.Domains,
		RateLimit: req.RateLimit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"setup_key":  setupKey.Token,
		"expires_at": formatTimePtr(setupKey.ExpiresAt),
		"scopes":     setupKey.Scopes,
		"server_url": fmt.Sprintf("http://127.0.0.1:%d", s.Port()),
	})
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !s.isRootRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Only the root token can mint sub-tokens"})
			return
		}
		var req struct {
			Action         string                `json:"action,omitempty"`
			ClientID       string                `json:"clientId"`
			Scopes         []tokenregistry.ScopeCategory `json:"scopes,omitempty"`
			Domains        []string              `json:"domains,omitempty"`
			TabPolicy      string                `json:"tabPolicy,omitempty"`
			RateLimit      int                   `json:"rateLimit,omitempty"`
			ExpiresSeconds *int                  `json:"expiresSeconds,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
			return
		}

		// Revoke action
		if req.Action == "revoke" || req.Action == "delete" {
			if req.ClientID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing clientId"})
				return
			}
			revoked := s.tokenRegistry.Revoke(req.ClientID)
			if !revoked {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Agent %q not found", req.ClientID)})
				return
			}
			log.Printf("[browse] Revoked token for: %s", req.ClientID)
			writeJSON(w, http.StatusOK, map[string]string{"revoked": req.ClientID})
			return
		}

		// Create token
		if req.ClientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing clientId"})
			return
		}

		session, err := s.tokenRegistry.CreateToken(tokenregistry.CreateTokenOptions{
			ClientID:       req.ClientID,
			Scopes:         req.Scopes,
			Domains:        req.Domains,
			TabPolicy:      req.TabPolicy,
			RateLimit:      req.RateLimit,
			ExpiresSeconds: req.ExpiresSeconds,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"token":  session.Token,
			"expires": formatTimePtr(session.ExpiresAt),
			"scopes": session.Scopes,
			"agent":  session.ClientID,
		})
		return
	}

	if r.Method == http.MethodDelete {
		if !s.isRootRequest(r) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Root token required"})
			return
		}
		clientID := strings.TrimPrefix(r.URL.Path, "/token/")
		if clientID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing clientId"})
			return
		}
		revoked := s.tokenRegistry.Revoke(clientID)
		if !revoked {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Agent %q not found", clientID)})
			return
		}
		log.Printf("[browse] Revoked token for: %s", clientID)
		writeJSON(w, http.StatusOK, map[string]string{"revoked": clientID})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.isRootRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Root token required"})
		return
	}

	tokens := s.tokenRegistry.ListTokens()
	agents := make([]map[string]interface{}, 0, len(tokens))
	for _, t := range tokens {
		agents = append(agents, map[string]interface{}{
			"clientId":     t.ClientID,
			"scopes":       t.Scopes,
			"domains":      t.Domains,
			"expiresAt":    formatTimePtr(t.ExpiresAt),
			"commandCount": t.CommandCount,
			"createdAt":    t.CreatedAt.Format(time.RFC3339Nano),
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"agents": agents})
}

// isRootRequest checks whether the request presents the root Bearer token.
func (s *Server) isRootRequest(r *http.Request) bool {
	token := extractBearer(r)
	return s.tokenRegistry.IsRoot(token)
}

// registryScopesToString converts tokenregistry scopes to a comma-separated
// string compatible with security.NewScopeSet.
func registryScopesToString(scopes []tokenregistry.ScopeCategory) string {
	var parts []string
	for _, s := range scopes {
		switch s {
		case tokenregistry.ScopeRead:
			parts = append(parts, "read")
		case tokenregistry.ScopeWrite:
			parts = append(parts, "interact", "write", "navigate")
		case tokenregistry.ScopeAdmin:
			parts = append(parts, "inspect", "write")
		case tokenregistry.ScopeMeta:
			parts = append(parts, "system")
		case tokenregistry.ScopeControl:
			parts = append(parts, "system")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// ─── Inspector Handlers ───────────────────────────────────

func (s *Server) handleInspectorPick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	selector := r.URL.Query().Get("selector")
	if selector == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing selector"})
		return
	}
	includeUA := r.URL.Query().Get("includeUA") == "true"

	ts, err := s.bm.GetActiveSession()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no active tab"})
		return
	}

	result, err := s.inspector.Inspect(ts.Context(), selector, includeUA)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleInspectorApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Selector string `json:"selector"`
		Property string `json:"property"`
		Value    string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Selector == "" || req.Property == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing selector or property"})
		return
	}

	ts, err := s.bm.GetActiveSession()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no active tab"})
		return
	}

	mod, err := s.inspector.Apply(ts.Context(), req.Selector, req.Property, req.Value)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, mod)
}

func (s *Server) handleInspectorReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ts, err := s.bm.GetActiveSession()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no active tab"})
		return
	}

	if err := s.inspector.Reset(ts.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "all modifications reset"})
}

func (s *Server) handleInspectorHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	history := s.inspector.History()
	writeJSON(w, http.StatusOK, map[string]interface{}{"history": history})
}

func (s *Server) handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.isRootRequest(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Root token required"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if tunnel is already active
	if s.tunnelActive && s.tunnelURL != "" {
		// Verify it's still alive
		if probeTunnel(s.tunnelURL) {
			writeJSON(w, http.StatusOK, map[string]interface{}{"url": s.tunnelURL, "already_active": true})
			return
		}
		// Dead — clean up
		if s.tunnel != nil {
			_ = s.tunnel.Close()
		}
		s.tunnel = nil
		s.tunnelActive = false
		s.tunnelURL = ""
	}

	authtoken := tunnel.ResolveAuthtoken()
	if authtoken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error": "No ngrok authtoken found",
			"hint":  "Run: ngrok config add-authtoken YOUR_TOKEN",
		})
		return
	}

	t, err := tunnel.Start(s.port, authtoken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.tunnel = t
	s.tunnelActive = true
	s.tunnelURL = t.URL()
	log.Printf("[browse] Tunnel started: %s", s.tunnelURL)

	writeJSON(w, http.StatusOK, map[string]interface{}{"url": s.tunnelURL})
}

func probeTunnel(url string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url + "/connect")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// StartTunnelSurface starts a secondary HTTP listener on an ephemeral port
// for the tunnel surface. It applies a restricted command allowlist.
func (s *Server) StartTunnelSurface() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("tunnel listener bind failed: %w", err)
	}

	tunnelMux := http.NewServeMux()
	// Only expose /connect and /command on the tunnel surface
	tunnelMux.HandleFunc("/connect", s.handleConnect)
	tunnelMux.HandleFunc("/command", s.handleTunnelCommand)

	go func() {
		_ = http.Serve(ln, s.withTunnelMiddleware(tunnelMux))
	}()

	return ln.Addr().(*net.TCPAddr).Port, nil
}

// withTunnelMiddleware wraps the handler with tunnel-specific restrictions:
//   - Root token is rejected (scoped tokens only)
//   - Command allowlist enforced
func (s *Server) withTunnelMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearer(r)
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		// Reject root token on tunnel
		if s.tokenRegistry.IsRoot(token) {
			log.Printf("[browse] Root token rejected on tunnel surface")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "root token not allowed over tunnel"})
			return
		}
		// Validate scoped token
		if info := s.tokenRegistry.Validate(token); info != nil {
			scopeStr := registryScopesToString(info.Scopes)
			r.Header.Set("X-Browse-Scope", scopeStr)
			r.Header.Set("X-Client-ID", info.ClientID)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (s *Server) handleTunnelCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	// Enforce tunnel allowlist
	if !tunnel.IsTunnelAllowed(req.Command) {
		log.Printf("[browse] Tunnel command denied: %s", req.Command)
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"ok":    false,
			"error": fmt.Sprintf("command %q not allowed over tunnel", req.Command),
		})
		return
	}

	opts := s.extractExecuteOpts(r)
	result, err := s.registry.ExecuteWithOpts(s.bm, req.Command, req.Args, opts)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "result": result})
}

func generateToken() string {
	b := make([]byte, 32)
	_, _ = randRead(b)
	return fmt.Sprintf("%x", b)
}

// readTerminalPort reads the terminal agent port from the state directory.
func (s *Server) readTerminalPort() int {
	cfg := config.Resolve(nil)
	port, _ := terminal.ReadPortFile(cfg.StateDir)
	return port
}

// handleWelcome serves the GStack Browser welcome page.
func (s *Server) handleWelcome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Try project-specific welcome page first
	slug := os.Getenv("GSTACK_SLUG")
	if slug == "" || !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(slug) {
		slug = "unknown"
	}
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		homeDir = "/tmp"
	}

	// Candidate paths
	var candidates []string
	projectWelcome := filepath.Join(homeDir, ".gstack", "projects", slug, "designs", "welcome-page-20260331", "finalized.html")
	candidates = append(candidates, projectWelcome)

	// Built-in welcome page (relative to repo root)
	if repoRoot := config.GitRoot(); repoRoot != "" {
		builtinWelcome := filepath.Join(repoRoot, "tools", "skills", "browse", "src", "welcome.html")
		candidates = append(candidates, builtinWelcome)
	}

	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(data)
			return
		}
	}

	// Fallback inline HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>GStack Browser</title>
<style>body{background:#111;color:#fff;font-family:system-ui;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;}
.msg{text-align:center;opacity:.7;}.gold{color:#f5a623;font-size:2em;margin-bottom:12px;}</style></head>
<body><div class="msg"><div class="gold">◈</div><p>GStack Browser ready.</p><p style="font-size:.85em">Waiting for commands from Claude Code.</p></div></body></html>`))
}

// handleSseSession mints an HttpOnly cookie for EventSource auth.
// The token is also returned in the JSON body so clients can use it via
// ?token= query parameter as a fallback.
func (s *Server) handleSseSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := generateToken()
	http.SetCookie(w, &http.Cookie{
		Name:     "gstack_sse",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600, // 1 hour
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":    true,
		"token": token,
	})
}

// handlePtySession mints a terminal WebSocket session token and pushes it to the agent.
func (s *Server) handlePtySession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := config.Resolve(nil)
	port, err := terminal.ReadPortFile(cfg.StateDir)
	if err != nil || port == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "terminal-agent not ready",
		})
		return
	}

	// Read internal token for loopback auth
	internalToken, err := terminal.ReadInternalTokenFile(cfg.StateDir)
	if err != nil || internalToken == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "terminal-agent internal token not found",
		})
		return
	}

	// Mint session token
	minted := generateToken()
	expiresAt := time.Now().Add(30 * time.Minute)

	// Push token to terminal agent
	grantBody, _ := json.Marshal(map[string]string{"token": minted})
	grantReq, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%d/internal/grant", port),
		bytes.NewReader(grantBody))
	grantReq.Header.Set("Authorization", "Bearer "+internalToken)
	grantReq.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	grantResp, err := client.Do(grantReq)
	if err != nil || grantResp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "failed to grant terminal session",
		})
		return
	}

	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws?token=%s", port, minted)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"terminalPort":    port,
		"ptySessionToken": minted,
		"expiresAt":       expiresAt.Format(time.RFC3339),
		"wsUrl":           wsURL,
	})
}

// handleInspectorEvents streams inspector modification events via SSE.
func (s *Server) handleInspectorEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Send current history as initial "snapshot" events
	for _, mod := range s.inspector.History() {
		writeInspectorSSE(w, inspector.Event{
			Type:      "apply",
			Timestamp: mod.Timestamp,
			Mod:       mod,
		})
	}

	// Subscribe to live inspector events
	ch := inspector.Subscribe()
	defer inspector.Unsubscribe(ch)

	// Block until client disconnects or server shuts down
	done := r.Context().Done()
	for {
		select {
		case <-done:
			return
		case e, ok := <-*ch:
			if !ok {
				return
			}
			writeInspectorSSE(w, e)
		}
	}
}

func writeInspectorSSE(w http.ResponseWriter, e inspector.Event) {
	data, _ := json.Marshal(e)
	fmt.Fprintf(w, "data: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// randRead is a thin shim so we can stub it in tests.
var randRead = func(b []byte) (int, error) {
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Read(b)
}
