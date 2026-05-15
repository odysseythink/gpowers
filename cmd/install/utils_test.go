package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home := homeDir()
	tests := []struct {
		input    string
		expected string
	}{
		{"~/foo", filepath.Join(home, "foo")},
		{"/abs/path", filepath.Clean("/abs/path")},
	}
	for _, tc := range tests {
		got := expandPath(tc.input)
		if got != tc.expected {
			t.Errorf("expandPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestCopyDir(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile a.txt failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0644); err != nil {
		t.Fatalf("WriteFile sub/b.txt failed: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil {
		t.Fatalf("ReadFile a.txt failed: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("a.txt content = %q, want %q", string(data), "hello")
	}
	data, err = os.ReadFile(filepath.Join(dst, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("ReadFile sub/b.txt failed: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("sub/b.txt content = %q, want %q", string(data), "world")
	}
}

func TestSameFile(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.txt")
	b := filepath.Join(tmp, "b.txt")
	if err := os.WriteFile(a, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile a.txt failed: %v", err)
	}
	if err := os.WriteFile(b, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile b.txt failed: %v", err)
	}
	if sameFile(a, b) {
		t.Error("sameFile should be false for different files")
	}
	if !sameFile(a, a) {
		t.Error("sameFile should be true for same file")
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"a", "b"}, "a") {
		t.Error("expected contains to find 'a'")
	}
	if contains([]string{"a", "b"}, "c") {
		t.Error("expected contains to not find 'c'")
	}
}
