package xvfb

import (
	"runtime"
	"testing"
)

func TestShouldSpawn(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		if ShouldSpawn() {
			t.Error("expected false on non-Linux")
		}
	}
}

func TestPickFreeDisplay(t *testing.T) {
	display, err := PickFreeDisplay()
	if err != nil {
		t.Fatalf("PickFreeDisplay failed: %v", err)
	}
	if display == "" {
		t.Error("expected non-empty display")
	}
	if display[0] != ':' {
		t.Errorf("expected display to start with :, got %s", display)
	}
}
