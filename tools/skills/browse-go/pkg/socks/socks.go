// Package socks implements a local SOCKS5 bridge that accepts unauthenticated
// connections and relays through an authenticated upstream SOCKS5 proxy.
//
// Architecture:
//
//	Chromium → socks5://127.0.0.1:<ephemeral> (this bridge, no auth)
//	           └→ authenticated SOCKS5 to upstream → destination
package socks

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// UpstreamConfig describes the authenticated upstream SOCKS5 proxy.
type UpstreamConfig struct {
	Host     string
	Port     int
	UserID   string
	Password string
}

// Bridge is a running local SOCKS5 listener.
type Bridge struct {
	Port      int
	listener  net.Listener
	inFlight  sync.Map // map[net.Conn]struct{}
	up        UpstreamConfig
	closeOnce sync.Once
	closed    chan struct{}
}

// SOCKS5 protocol constants.
const (
	ver5              = 0x05
	cmdConnect        = 0x01
	atypIPv4          = 0x01
	atypDomainName    = 0x03
	atypIPv6          = 0x04
	repSuccess        = 0x00
	repGeneralFailure = 0x01
	repHostUnreachable = 0x04
	authNone          = 0x00
	authPassword      = 0x02
	authNoAcceptable  = 0xff
)

// StartBridge creates a local SOCKS5 listener on 127.0.0.1 (port 0 = ephemeral).
func StartBridge(upstream UpstreamConfig, port int) (*Bridge, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("socks bridge listen: %w", err)
	}

	b := &Bridge{
		Port:     ln.Addr().(*net.TCPAddr).Port,
		listener: ln,
		up:       upstream,
		closed:   make(chan struct{}),
	}

	go b.serve()
	return b, nil
}

// Close shuts down the bridge and all in-flight connections.
func (b *Bridge) Close() error {
	var firstErr error
	b.closeOnce.Do(func() {
		firstErr = b.listener.Close()
		close(b.closed)
		b.inFlight.Range(func(key, _ any) bool {
			if conn, ok := key.(net.Conn); ok {
				conn.Close()
			}
			return true
		})
	})
	return firstErr
}

func (b *Bridge) serve() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			select {
			case <-b.closed:
				return
			default:
				continue
			}
		}
		b.inFlight.Store(conn, struct{}{})
		go b.handle(conn)
	}
}

func (b *Bridge) handle(client net.Conn) {
	defer func() {
		client.Close()
		b.inFlight.Delete(client)
	}()

	// Handshake timeout
	client.SetDeadline(time.Now().Add(30 * time.Second))
	defer client.SetDeadline(time.Time{})

	// 1. Greeting
	if err := b.handshake(client); err != nil {
		return
	}

	// 2. CONNECT request
	destHost, destPort, err := b.readConnectRequest(client)
	if err != nil {
		writeReply(client, repGeneralFailure)
		return
	}

	// 3. Dial upstream SOCKS5 proxy
	upAddr := fmt.Sprintf("%s:%d", b.up.Host, b.up.Port)
	upConn, err := net.DialTimeout("tcp", upAddr, 15*time.Second)
	if err != nil {
		writeReply(client, repHostUnreachable)
		return
	}
	defer upConn.Close()

	// 4. SOCKS5 handshake with upstream (with auth if configured)
	if err := upstreamHandshake(upConn, b.up); err != nil {
		writeReply(client, repGeneralFailure)
		return
	}

	// 5. Send CONNECT through upstream
	if err := upstreamConnect(upConn, destHost, destPort); err != nil {
		writeReply(client, repHostUnreachable)
		return
	}

	// 6. Reply success to client
	if err := writeReply(client, repSuccess); err != nil {
		return
	}

	// 7. Bidirectional pipe
	client.SetDeadline(time.Time{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(upConn, client)
		upConn.(*net.TCPConn).CloseWrite()
	}()
	go func() {
		defer wg.Done()
		io.Copy(client, upConn)
		client.(*net.TCPConn).CloseWrite()
	}()
	wg.Wait()
}

// handshake reads the client greeting and replies with no-auth method.
func (b *Bridge) handshake(client net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(client, buf); err != nil {
		return err
	}
	if buf[0] != ver5 {
		return fmt.Errorf("unsupported SOCKS version: 0x%02x", buf[0])
	}
	nmethods := int(buf[1])
	methods := make([]byte, nmethods)
	if _, err := io.ReadFull(client, methods); err != nil {
		return err
	}
	// We only support no-auth for the local side
	_, err := client.Write([]byte{ver5, authNone})
	return err
}

// readConnectRequest parses a SOCKS5 CONNECT request.
func (b *Bridge) readConnectRequest(client net.Conn) (host string, port int, err error) {
	header := make([]byte, 4)
	if _, err = io.ReadFull(client, header); err != nil {
		return "", 0, err
	}
	if header[0] != ver5 || header[1] != cmdConnect {
		return "", 0, fmt.Errorf("bad CONNECT request")
	}

	atyp := header[3]
	switch atyp {
	case atypIPv4:
		addr := make([]byte, 4)
		if _, err = io.ReadFull(client, addr); err != nil {
			return "", 0, err
		}
		host = net.IP(addr).String()
	case atypDomainName:
		lenBuf := make([]byte, 1)
		if _, err = io.ReadFull(client, lenBuf); err != nil {
			return "", 0, err
		}
		domain := make([]byte, int(lenBuf[0]))
		if _, err = io.ReadFull(client, domain); err != nil {
			return "", 0, err
		}
		host = string(domain)
	case atypIPv6:
		addr := make([]byte, 16)
		if _, err = io.ReadFull(client, addr); err != nil {
			return "", 0, err
		}
		host = net.IP(addr).String()
	default:
		return "", 0, fmt.Errorf("unsupported ATYP: 0x%02x", atyp)
	}

	portBuf := make([]byte, 2)
	if _, err = io.ReadFull(client, portBuf); err != nil {
		return "", 0, err
	}
	port = int(binary.BigEndian.Uint16(portBuf))
	return host, port, nil
}

func writeReply(w io.Writer, code byte) error {
	// VER REP RSV ATYP BND.ADDR(0.0.0.0) BND.PORT(0)
	_, err := w.Write([]byte{ver5, code, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
	return err
}

// ─── Upstream SOCKS5 client ─────────────────────────────────────

func upstreamHandshake(conn net.Conn, up UpstreamConfig) error {
	// Offer auth methods
	if up.UserID != "" && up.Password != "" {
		_, err := conn.Write([]byte{ver5, 2, authNone, authPassword})
		if err != nil {
			return err
		}
		resp := make([]byte, 2)
		if _, err := io.ReadFull(conn, resp); err != nil {
			return err
		}
		if resp[0] != ver5 {
			return errors.New("upstream SOCKS5 version mismatch")
		}
		if resp[1] == authPassword {
			return upstreamPasswordAuth(conn, up.UserID, up.Password)
		}
		if resp[1] == authNone {
			return nil
		}
		return errors.New("upstream rejected all auth methods")
	}

	// No auth required
	_, err := conn.Write([]byte{ver5, 1, authNone})
	if err != nil {
		return err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if resp[0] != ver5 || resp[1] != authNone {
		return errors.New("upstream rejected no-auth method")
	}
	return nil
}

func upstreamPasswordAuth(conn net.Conn, user, pass string) error {
	// VER(0x01) ULEN USER PLEN PASS
	ub := []byte(user)
	pb := []byte(pass)
	buf := make([]byte, 0, 3+len(ub)+len(pb))
	buf = append(buf, 0x01, byte(len(ub)))
	buf = append(buf, ub...)
	buf = append(buf, byte(len(pb)))
	buf = append(buf, pb...)
	if _, err := conn.Write(buf); err != nil {
		return err
	}
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if resp[0] != 0x01 || resp[1] != 0x00 {
		return errors.New("upstream SOCKS5 auth failed")
	}
	return nil
}

func upstreamConnect(conn net.Conn, host string, port int) error {
	req := buildConnectRequest(host, port)
	if _, err := conn.Write(req); err != nil {
		return err
	}
	resp := make([]byte, 4)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return err
	}
	if resp[0] != ver5 || resp[1] != repSuccess {
		return fmt.Errorf("upstream CONNECT failed: code 0x%02x", resp[1])
	}
	// Skip BND.ADDR + BND.PORT
	atyp := resp[3]
	switch atyp {
	case atypIPv4:
		if _, err := io.ReadFull(conn, make([]byte, 6)); err != nil {
			return err
		}
	case atypIPv6:
		if _, err := io.ReadFull(conn, make([]byte, 18)); err != nil {
			return err
		}
	case atypDomainName:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return err
		}
		if _, err := io.ReadFull(conn, make([]byte, int(lenBuf[0])+2)); err != nil {
			return err
		}
	}
	return nil
}

func buildConnectRequest(host string, port int) []byte {
	ip := net.ParseIP(host)
	var req []byte
	req = append(req, ver5, cmdConnect, 0x00)
	if ip4 := ip.To4(); ip4 != nil {
		req = append(req, atypIPv4)
		req = append(req, ip4...)
	} else if ip16 := ip.To16(); ip16 != nil {
		req = append(req, atypIPv6)
		req = append(req, ip16...)
	} else {
		req = append(req, atypDomainName, byte(len(host)))
		req = append(req, host...)
	}
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, uint16(port))
	req = append(req, portBuf...)
	return req
}

// ─── TestUpstream ───────────────────────────────────────────────

// TestResult is returned by TestUpstream.
type TestResult struct {
	OK       bool
	Attempts int
	MS       int
}

// TestUpstream verifies the upstream proxy accepts credentials and can reach
// a known endpoint. Retries with backoff for residential VPN warm-up.
func TestUpstream(up UpstreamConfig, testHost string, testPort int) (*TestResult, error) {
	if testHost == "" {
		testHost = "1.1.1.1"
	}
	if testPort == 0 {
		testPort = 443
	}
	budget := 5 * time.Second
	retries := 3
	backoff := 500 * time.Millisecond

	start := time.Now()
	var lastErr error

	for attempt := 1; attempt <= retries; attempt++ {
		elapsed := time.Since(start)
		remaining := budget - elapsed
		if remaining <= 0 {
			break
		}
		perAttempt := min(remaining, max(500*time.Millisecond, budget/time.Duration(retries)))

		ctx, cancel := context.WithTimeout(context.Background(), perAttempt)
		dialer := &net.Dialer{Timeout: perAttempt}
		upAddr := fmt.Sprintf("%s:%d", up.Host, up.Port)
		conn, err := dialer.DialContext(ctx, "tcp", upAddr)
		cancel()
		if err != nil {
			lastErr = err
			if attempt < retries && time.Since(start)+backoff < budget {
				time.Sleep(backoff)
			}
			continue
		}

		if err := upstreamHandshake(conn, up); err != nil {
			conn.Close()
			lastErr = err
			if attempt < retries && time.Since(start)+backoff < budget {
				time.Sleep(backoff)
			}
			continue
		}

		if err := upstreamConnect(conn, testHost, testPort); err != nil {
			conn.Close()
			lastErr = err
			if attempt < retries && time.Since(start)+backoff < budget {
				time.Sleep(backoff)
			}
			continue
		}

		conn.Close()
		return &TestResult{OK: true, Attempts: attempt, MS: int(time.Since(start).Milliseconds())}, nil
	}

	return nil, fmt.Errorf("SOCKS5 upstream rejected or unreachable after %d attempts (%dms): %v",
		retries, int(time.Since(start).Milliseconds()), lastErr)
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
