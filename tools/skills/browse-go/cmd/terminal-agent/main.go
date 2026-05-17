// terminal-agent — WebSocket-to-PTY bridge for the browse sidebar Terminal pane.
//
// Spawns a shell and bridges WebSocket binary frames to PTY I/O.
// The parent browse server pushes session tokens via POST /internal/grant.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"browse-go/pkg/config"
	"browse-go/pkg/terminal"
)

func main() {
	log.SetFlags(0)

	cfg := config.Resolve(nil)
	agent := terminal.NewAgent()

	// Read parent server port from env
	if p := os.Getenv("BROWSE_SERVER_PORT"); p != "" {
		var port int
		if _, err := fmt.Sscanf(p, "%d", &port); err == nil {
			agent.SetPort(port)
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("[terminal-agent] failed to bind: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	// Write port file atomically
	if err := terminal.WritePortFile(cfg.StateDir, port); err != nil {
		log.Printf("[terminal-agent] warning: cannot write port file: %v", err)
	}

	// Write internal token file
	if err := terminal.WriteInternalTokenFile(cfg.StateDir, agent.InternalToken()); err != nil {
		log.Printf("[terminal-agent] warning: cannot write token file: %v", err)
	}

	log.Printf("[terminal-agent] listening on 127.0.0.1:%d", port)

	// Signal cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		terminal.RemovePortFile(cfg.StateDir)
		terminal.RemoveInternalTokenFile(cfg.StateDir)
		os.Exit(0)
	}()

	server := &http.Server{Handler: agent.Handler()}
	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[terminal-agent] server error: %v", err)
	}
}
