//go:build !windows

// Package terminal implements a WebSocket-to-PTY bridge for the browse sidebar
// Terminal pane. Spawns a shell (or configured binary) and bridges binary
// frames to stdin/stdout.
package terminal

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/kr/pty"
)

// Agent is a WebSocket server that bridges to PTY sessions.
type Agent struct {
	mu           sync.RWMutex
	upgrader     websocket.Upgrader
	sessions     map[*websocket.Conn]*session
	validTokens  map[string]bool
	internalToken string
	port         int
}

type session struct {
	pty      *os.File
	cmd      *exec.Cmd
	cols     int
	rows     int
	cookie   string
	spawned  bool
	mu       sync.Mutex
}

// NewAgent creates a terminal agent.
func NewAgent() *Agent {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return &Agent{
		sessions:      make(map[*websocket.Conn]*session),
		validTokens:   make(map[string]bool),
		internalToken: hex.EncodeToString(b),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				return strings.HasPrefix(origin, "chrome-extension://") || origin == ""
			},
			Subprotocols: []string{},
		},
	}
}

// InternalToken returns the parent-only auth token.
func (a *Agent) InternalToken() string {
	return a.internalToken
}

// Grant registers a user session token.
func (a *Agent) Grant(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(token) > 16 {
		a.validTokens[token] = true
	}
}

// Revoke removes a user session token.
func (a *Agent) Revoke(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.validTokens, token)
}

// Handler returns the HTTP handler for the agent.
func (a *Agent) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/grant", a.handleGrant)
	mux.HandleFunc("/internal/revoke", a.handleRevoke)
	mux.HandleFunc("/ws", a.handleWS)
	return mux
}

func (a *Agent) handleGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+a.internalToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Token == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	a.Grant(body.Token)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *Agent) handleRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	auth := r.Header.Get("Authorization")
	if auth != "Bearer "+a.internalToken {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Token == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	a.Revoke(body.Token)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *Agent) handleWS(w http.ResponseWriter, r *http.Request) {
	// Extract token from Sec-WebSocket-Protocol or Cookie
	token := ""
	proto := r.Header.Get("Sec-WebSocket-Protocol")
	for _, raw := range strings.Split(proto, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		candidate := raw
		if strings.HasPrefix(raw, "gstack-pty.") {
			candidate = raw[len("gstack-pty."):]
		}
		if a.isValidToken(candidate) {
			token = candidate
			break
		}
	}
	if token == "" {
		cookie := r.Header.Get("Cookie")
		for _, part := range strings.Split(cookie, ";") {
			part = strings.TrimSpace(part)
			if name, val, ok := strings.Cut(part, "="); ok && name == "gstack_pty" {
				if a.isValidToken(val) {
					token = val
					break
				}
			}
		}
	}
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Set subprotocol if matched via header
	if proto != "" && token != "" {
		a.upgrader.Subprotocols = []string{proto}
	}

	ws, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[terminal-agent] ws upgrade failed: %v", err)
		return
	}

	sess := &session{cols: 80, rows: 24, cookie: token}
	a.mu.Lock()
	a.sessions[ws] = sess
	a.mu.Unlock()

	// Read loop
	go a.readLoop(ws, sess)
}

func (a *Agent) readLoop(ws *websocket.Conn, sess *session) {
	defer func() {
		a.mu.Lock()
		delete(a.sessions, ws)
		a.mu.Unlock()
		if sess.pty != nil {
			_ = sess.pty.Close()
		}
		if sess.cmd != nil && sess.cmd.Process != nil {
			_ = sess.cmd.Process.Kill()
		}
		_ = ws.Close()
	}()

	for {
		msgType, data, err := ws.ReadMessage()
		if err != nil {
			return
		}

		if msgType == websocket.TextMessage {
			// Control messages
			var ctrl struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if err := decodeBytes(data, &ctrl); err == nil {
				if ctrl.Type == "resize" && ctrl.Cols > 0 && ctrl.Rows > 0 {
					sess.mu.Lock()
					sess.cols = ctrl.Cols
					sess.rows = ctrl.Rows
					if sess.pty != nil {
						_ = pty.Setsize(sess.pty, &pty.Winsize{Cols: uint16(ctrl.Cols), Rows: uint16(ctrl.Rows)})
					}
					sess.mu.Unlock()
				}
			}
			continue
		}

		if msgType == websocket.BinaryMessage {
			if !sess.spawned {
				sess.spawned = true
				if err := a.spawn(sess, ws); err != nil {
					log.Printf("[terminal-agent] spawn failed: %v", err)
					return
				}
			}
			sess.mu.Lock()
			if sess.pty != nil {
				_, _ = sess.pty.Write(data)
			}
			sess.mu.Unlock()
		}
	}
}

func (a *Agent) spawn(sess *session, ws *websocket.Conn) error {
	shell := defaultShell()
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		fmt.Sprintf("BROWSE_PORT=%d", a.port),
	)
	if sf := os.Getenv("BROWSE_STATE_FILE"); sf != "" {
		cmd.Env = append(cmd.Env, "BROWSE_STATE_FILE="+sf)
		cmd.Env = append(cmd.Env, "BROWSE_STATE_DIR="+filepath.Dir(sf))
	}
	if p := os.Getenv("BROWSE_SERVER_PORT"); p != "" {
		cmd.Env = append(cmd.Env, "BROWSE_SERVER_PORT="+p)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	sess.mu.Lock()
	sess.pty = ptmx
	sess.cmd = cmd
	sess.mu.Unlock()

	_ = pty.Setsize(ptmx, &pty.Winsize{Cols: uint16(sess.cols), Rows: uint16(sess.rows)})

	// PTY output → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Wait for process exit
	go func() {
		_ = cmd.Wait()
		_ = ws.Close()
	}()

	return nil
}

func (a *Agent) isValidToken(token string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.validTokens[token]
}

// SetPort records the parent browse server port for env injection.
func (a *Agent) SetPort(port int) {
	a.port = port
}

func defaultShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/bash"
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func decodeBytes(data []byte, v interface{}) error {
	return json.NewDecoder(bytes.NewReader(data)).Decode(v)
}
