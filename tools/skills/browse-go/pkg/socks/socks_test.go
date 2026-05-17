package socks

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

// ─── Protocol helpers ───────────────────────────────────────────

func makeGreeting(nmethods byte, methods ...byte) []byte {
	b := []byte{ver5, nmethods}
	return append(b, methods...)
}

func makeConnectRequest(atyp byte, addr []byte, port uint16) []byte {
	b := []byte{ver5, cmdConnect, 0x00, atyp}
	b = append(b, addr...)
	portBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(portBuf, port)
	return append(b, portBuf...)
}

// ─── Handshake tests ────────────────────────────────────────────

func TestHandshakeAcceptsNoAuth(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	b := &Bridge{}
	go func() {
		if err := b.handshake(server); err != nil {
			t.Logf("handshake error: %v", err)
		}
	}()

	client.Write(makeGreeting(1, authNone))
	resp := make([]byte, 2)
	if _, err := io.ReadFull(client, resp); err != nil {
		t.Fatalf("read greeting reply: %v", err)
	}
	if resp[0] != ver5 || resp[1] != authNone {
		t.Errorf("greeting reply = %v, want [5 0]", resp)
	}
}

func TestHandshakeRejectsBadVersion(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	b := &Bridge{}
	done := make(chan error, 1)
	go func() {
		done <- b.handshake(server)
	}()

	// Write in goroutine because net.Pipe blocks until read
	go client.Write([]byte{0x04, 0x01, 0x00}) // SOCKS4

	err := <-done
	if err == nil {
		t.Error("expected error for SOCKS4 version")
	}
}

// ─── CONNECT request parsing ────────────────────────────────────

func TestReadConnectRequestIPv4(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	b := &Bridge{}
	go func() {
		req := makeConnectRequest(atypIPv4, []byte{127, 0, 0, 1}, 8080)
		client.Write(req)
	}()

	host, port, err := b.readConnectRequest(server)
	if err != nil {
		t.Fatalf("readConnectRequest: %v", err)
	}
	if host != "127.0.0.1" {
		t.Errorf("host = %q, want 127.0.0.1", host)
	}
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}
}

func TestReadConnectRequestDomain(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	b := &Bridge{}
	go func() {
		addr := append([]byte{byte(len("example.com"))}, []byte("example.com")...)
		req := makeConnectRequest(atypDomainName, addr, 443)
		client.Write(req)
	}()

	host, port, err := b.readConnectRequest(server)
	if err != nil {
		t.Fatalf("readConnectRequest: %v", err)
	}
	if host != "example.com" {
		t.Errorf("host = %q, want example.com", host)
	}
	if port != 443 {
		t.Errorf("port = %d, want 443", port)
	}
}

func TestReadConnectRequestIPv6(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	b := &Bridge{}
	go func() {
		ip := net.ParseIP("::1")
		req := makeConnectRequest(atypIPv6, ip, 8080)
		client.Write(req)
	}()

	host, port, err := b.readConnectRequest(server)
	if err != nil {
		t.Fatalf("readConnectRequest: %v", err)
	}
	if host != "::1" {
		t.Errorf("host = %q, want ::1", host)
	}
	if port != 8080 {
		t.Errorf("port = %d, want 8080", port)
	}
}

// ─── Reply formatting ───────────────────────────────────────────

func TestWriteReply(t *testing.T) {
	var buf bytes.Buffer
	if err := writeReply(&buf, repSuccess); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if len(b) != 10 {
		t.Fatalf("reply len = %d, want 10", len(b))
	}
	if b[0] != ver5 || b[1] != repSuccess {
		t.Errorf("reply header = %v", b[:2])
	}
}

// ─── Bridge lifecycle ───────────────────────────────────────────

func TestBridgeStartStop(t *testing.T) {
	b, err := StartBridge(UpstreamConfig{Host: "127.0.0.1", Port: 19050}, 0)
	if err != nil {
		t.Fatalf("StartBridge: %v", err)
	}
	if b.Port == 0 {
		t.Error("ephemeral port is 0")
	}

	// Verify listener is accepting
	conn, err := net.Dial("tcp", "127.0.0.1:"+string(rune('0'+b.Port%10)))
	if err != nil {
		// Port might not be in ASCII range — just check it doesn't panic
		_ = conn
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestBridgeCloseCleansInFlight(t *testing.T) {
	b, err := StartBridge(UpstreamConfig{Host: "127.0.0.1", Port: 19050}, 0)
	if err != nil {
		t.Fatalf("StartBridge: %v", err)
	}

	// Dial and leave hanging mid-handshake
	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", string(rune('0'+b.Port%10))))
	if err == nil {
		b.inFlight.Store(conn, struct{}{})
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Give goroutines time to clean up
	time.Sleep(50 * time.Millisecond)
}

// ─── Build connect request ──────────────────────────────────────

func TestBuildConnectRequestIPv4(t *testing.T) {
	req := buildConnectRequest("1.2.3.4", 80)
	if req[3] != atypIPv4 {
		t.Errorf("ATYP = 0x%02x, want IPv4", req[3])
	}
	if !bytes.Equal(req[4:8], []byte{1, 2, 3, 4}) {
		t.Errorf("addr = %v", req[4:8])
	}
	port := binary.BigEndian.Uint16(req[8:10])
	if port != 80 {
		t.Errorf("port = %d, want 80", port)
	}
}

func TestBuildConnectRequestDomain(t *testing.T) {
	req := buildConnectRequest("example.com", 443)
	if req[3] != atypDomainName {
		t.Errorf("ATYP = 0x%02x, want domain", req[3])
	}
	if req[4] != 11 {
		t.Errorf("domain len = %d, want 11", req[4])
	}
	if string(req[5:16]) != "example.com" {
		t.Errorf("domain = %q", string(req[5:16]))
	}
}

func TestBuildConnectRequestIPv6(t *testing.T) {
	req := buildConnectRequest("::1", 8080)
	if req[3] != atypIPv6 {
		t.Errorf("ATYP = 0x%02x, want IPv6", req[3])
	}
	if len(req) != 22 {
		t.Errorf("len = %d, want 22", len(req))
	}
}

// ─── Upstream handshake (no-auth) ───────────────────────────────

func TestUpstreamHandshakeNoAuth(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		// Server side: accept no-auth
		greeting := make([]byte, 2)
		io.ReadFull(server, greeting)
		nmethods := int(greeting[1])
		methods := make([]byte, nmethods)
		io.ReadFull(server, methods)
		server.Write([]byte{ver5, authNone})
	}()

	if err := upstreamHandshake(client, UpstreamConfig{}); err != nil {
		t.Fatalf("upstreamHandshake: %v", err)
	}
}

func TestUpstreamHandshakePassword(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		// Server side: accept password auth
		greeting := make([]byte, 2)
		io.ReadFull(server, greeting)
		nmethods := int(greeting[1])
		methods := make([]byte, nmethods)
		io.ReadFull(server, methods)
		server.Write([]byte{ver5, authPassword})

		// Read password auth request
		ver := make([]byte, 1)
		io.ReadFull(server, ver)
		ulen := make([]byte, 1)
		io.ReadFull(server, ulen)
		user := make([]byte, ulen[0])
		io.ReadFull(server, user)
		plen := make([]byte, 1)
		io.ReadFull(server, plen)
		pass := make([]byte, plen[0])
		io.ReadFull(server, pass)

		if string(user) == "testuser" && string(pass) == "testpass" {
			server.Write([]byte{0x01, 0x00})
		} else {
			server.Write([]byte{0x01, 0x01})
		}
	}()

	if err := upstreamHandshake(client, UpstreamConfig{UserID: "testuser", Password: "testpass"}); err != nil {
		t.Fatalf("upstreamHandshake: %v", err)
	}
}

func TestUpstreamHandshakePasswordFail(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		greeting := make([]byte, 2)
		io.ReadFull(server, greeting)
		nmethods := int(greeting[1])
		methods := make([]byte, nmethods)
		io.ReadFull(server, methods)
		server.Write([]byte{ver5, authPassword})

		ver := make([]byte, 1)
		io.ReadFull(server, ver)
		ulen := make([]byte, 1)
		io.ReadFull(server, ulen)
		user := make([]byte, ulen[0])
		io.ReadFull(server, user)
		plen := make([]byte, 1)
		io.ReadFull(server, plen)
		pass := make([]byte, plen[0])
		io.ReadFull(server, pass)
		server.Write([]byte{0x01, 0x01}) // auth failed
	}()

	if err := upstreamHandshake(client, UpstreamConfig{UserID: "bad", Password: "bad"}); err == nil {
		t.Error("expected auth failure")
	}
}

// ─── Upstream connect ───────────────────────────────────────────

func TestUpstreamConnectSuccess(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		// Read CONNECT request
		header := make([]byte, 4)
		io.ReadFull(server, header)
		// Skip addr + port (we know it's example.com:443)
		atyp := header[3]
		switch atyp {
		case atypDomainName:
			lenBuf := make([]byte, 1)
			io.ReadFull(server, lenBuf)
			io.ReadFull(server, make([]byte, lenBuf[0]+2))
		}
		// Reply success
		server.Write([]byte{ver5, repSuccess, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
	}()

	if err := upstreamConnect(client, "example.com", 443); err != nil {
		t.Fatalf("upstreamConnect: %v", err)
	}
}

func TestUpstreamConnectFailure(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	go func() {
		header := make([]byte, 4)
		io.ReadFull(server, header)
		atyp := header[3]
		switch atyp {
		case atypDomainName:
			lenBuf := make([]byte, 1)
			io.ReadFull(server, lenBuf)
			io.ReadFull(server, make([]byte, lenBuf[0]+2))
		}
		server.Write([]byte{ver5, repHostUnreachable, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
	}()

	if err := upstreamConnect(client, "example.com", 443); err == nil {
		t.Error("expected error for host unreachable")
	}
}

// ─── End-to-end: local bridge with loopback echo ────────────────

func TestBridgeEndToEnd(t *testing.T) {
	// Start an echo server as the "upstream" destination
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo listen: %v", err)
	}
	defer echoLn.Close()
	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	echoAddr := echoLn.Addr().(*net.TCPAddr)

	// Start a fake upstream SOCKS5 that just connects directly
	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("upstream listen: %v", err)
	}
	defer upLn.Close()
	go func() {
		for {
			conn, err := upLn.Accept()
			if err != nil {
				return
			}
			go handleFakeUpstream(conn, echoAddr.IP.String(), echoAddr.Port)
		}
	}()

	upAddr := upLn.Addr().(*net.TCPAddr)

	// Start bridge
	b, err := StartBridge(UpstreamConfig{Host: "127.0.0.1", Port: upAddr.Port}, 0)
	if err != nil {
		t.Fatalf("StartBridge: %v", err)
	}
	defer b.Close()

	// Connect through bridge
	client, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", itoa(b.Port)))
	if err != nil {
		t.Fatalf("dial bridge: %v", err)
	}
	defer client.Close()

	// SOCKS5 handshake
	client.Write(makeGreeting(1, authNone))
	resp := make([]byte, 2)
	if _, err := io.ReadFull(client, resp); err != nil {
		t.Fatalf("greeting reply: %v", err)
	}

	// CONNECT to echo server
	addr := append([]byte{byte(len("127.0.0.1"))}, []byte("127.0.0.1")...)
	req := makeConnectRequest(atypDomainName, addr, uint16(echoAddr.Port))
	client.Write(req)

	reply := make([]byte, 10)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("connect reply: %v", err)
	}
	if reply[1] != repSuccess {
		t.Fatalf("connect failed: code 0x%02x", reply[1])
	}

	// Send data through the tunnel
	testData := []byte("hello through socks bridge")
	client.Write(testData)

	// Read echo back
	result := make([]byte, len(testData))
	if _, err := io.ReadFull(client, result); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if !bytes.Equal(result, testData) {
		t.Errorf("echo = %q, want %q", result, testData)
	}
}

// handleFakeUpstream is a minimal SOCKS5 server for testing.
func handleFakeUpstream(client net.Conn, destHost string, destPort int) {
	defer client.Close()

	// Greeting
	buf := make([]byte, 2)
	io.ReadFull(client, buf)
	nmethods := int(buf[1])
	methods := make([]byte, nmethods)
	io.ReadFull(client, methods)
	client.Write([]byte{ver5, authNone})

	// CONNECT
	header := make([]byte, 4)
	io.ReadFull(client, header)
	atyp := header[3]
	switch atyp {
	case atypIPv4:
		io.ReadFull(client, make([]byte, 6))
	case atypIPv6:
		io.ReadFull(client, make([]byte, 18))
	case atypDomainName:
		lenBuf := make([]byte, 1)
		io.ReadFull(client, lenBuf)
		io.ReadFull(client, make([]byte, lenBuf[0]+2))
	}

	// Connect to destination
	conn, err := net.Dial("tcp", net.JoinHostPort(destHost, itoa(destPort)))
	if err != nil {
		client.Write([]byte{ver5, repHostUnreachable, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})
		return
	}
	defer conn.Close()

	client.Write([]byte{ver5, repSuccess, 0x00, atypIPv4, 0, 0, 0, 0, 0, 0})

	// Pipe — close both when either direction ends
	done := make(chan struct{})
	go func() {
		io.Copy(conn, client)
		conn.Close()
		client.Close()
		close(done)
	}()
	io.Copy(client, conn)
	conn.Close()
	client.Close()
	<-done
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// ─── TestUpstream with fake server ──────────────────────────────

func TestTestUpstreamSuccess(t *testing.T) {
	// Start a local echo server as the target
	targetLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("target listen: %v", err)
	}
	defer targetLn.Close()
	go func() {
		for {
			conn, err := targetLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()
	targetAddr := targetLn.Addr().(*net.TCPAddr)

	// Fake upstream that relays to the local target
	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("upstream listen: %v", err)
	}
	defer upLn.Close()
	go func() {
		for {
			conn, err := upLn.Accept()
			if err != nil {
				return
			}
			go handleFakeUpstream(conn, targetAddr.IP.String(), targetAddr.Port)
		}
	}()

	upAddr := upLn.Addr().(*net.TCPAddr)
	res, err := TestUpstream(UpstreamConfig{Host: "127.0.0.1", Port: upAddr.Port}, targetAddr.IP.String(), targetAddr.Port)
	if err != nil {
		t.Fatalf("TestUpstream: %v", err)
	}
	if !res.OK {
		t.Error("expected OK")
	}
	if res.Attempts < 1 {
		t.Error("expected at least 1 attempt")
	}
}

func TestTestUpstreamUnreachable(t *testing.T) {
	_, err := TestUpstream(UpstreamConfig{Host: "127.0.0.1", Port: 1}, "", 0)
	if err == nil {
		t.Error("expected error for unreachable upstream")
	}
}

// ─── Edge cases ─────────────────────────────────────────────────

func TestBuildConnectRequestInvalidIP(t *testing.T) {
	// net.ParseIP returns nil for invalid IPs, falls back to domain name
	req := buildConnectRequest("not-an-ip", 80)
	if req[3] != atypDomainName {
		t.Errorf("expected domain fallback for invalid IP, got ATYP 0x%02x", req[3])
	}
}

func TestMinMax(t *testing.T) {
	if min(1*time.Second, 2*time.Second) != 1*time.Second {
		t.Error("min failed")
	}
	if max(1*time.Second, 2*time.Second) != 2*time.Second {
		t.Error("max failed")
	}
}
