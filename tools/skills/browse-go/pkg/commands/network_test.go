package commands

import (
	"strings"
	"testing"
)

func TestRegistryNetworkCommandFlags(t *testing.T) {
	r := NewRegistry()
	desc, ok := r.GetDesc("network")
	if !ok {
		t.Fatal("expected network command to be registered")
	}
	usage := desc.Usage
	for _, flag := range []string{"--clear", "--capture", "--filter", "--export", "--bodies"} {
		if !strings.Contains(usage, flag) {
			t.Errorf("expected network usage to contain %q, got %q", flag, usage)
		}
	}
}
