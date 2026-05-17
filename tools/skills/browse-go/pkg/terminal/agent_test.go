package terminal

import (
	"testing"
)

func TestGrantRevoke(t *testing.T) {
	a := NewAgent()
	longToken := "this-is-a-valid-session-token-xyz"
	if a.isValidToken(longToken) {
		t.Error("expected token invalid before grant")
	}
	a.Grant(longToken)
	if !a.isValidToken(longToken) {
		t.Error("expected token valid after grant")
	}
	a.Revoke(longToken)
	if a.isValidToken(longToken) {
		t.Error("expected token invalid after revoke")
	}
}

func TestGrantMinLength(t *testing.T) {
	a := NewAgent()
	a.Grant("short")
	if a.isValidToken("short") {
		t.Error("expected short token not granted")
	}
}

func TestInternalToken(t *testing.T) {
	a := NewAgent()
	if a.InternalToken() == "" {
		t.Error("expected non-empty internal token")
	}
}
