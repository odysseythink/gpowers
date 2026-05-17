package commands

import (
	"testing"
)

func TestUniqueStrings(t *testing.T) {
	in := []string{"a", "b", "a", "c", "b", "d"}
	out := uniqueStrings(in)
	if len(out) != 4 {
		t.Fatalf("expected 4 unique, got %d", len(out))
	}
	expected := []string{"a", "b", "c", "d"}
	for i, v := range expected {
		if out[i] != v {
			t.Errorf("expected %q at %d, got %q", v, i, out[i])
		}
	}
}

func TestUniqueStringsEmpty(t *testing.T) {
	out := uniqueStrings(nil)
	if len(out) != 0 {
		t.Fatalf("expected empty, got %v", out)
	}
}
