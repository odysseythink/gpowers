package platform

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsPathWithin(t *testing.T) {
	sep := string(filepath.Separator)
	if !IsPathWithin("/foo"+sep+"bar", "/foo") {
		t.Fatal("expected within")
	}
	if IsPathWithin("/foo"+sep+"bar", "/foo"+sep+"baz") {
		t.Fatal("expected not within")
	}
	if !IsPathWithin("/foo", "/foo") {
		t.Fatal("expected equal path to be within")
	}
	if runtime.GOOS == "windows" {
		if !IsPathWithin(`C:\Users\foo`, `C:\Users`) {
			t.Fatal("expected windows path within")
		}
	}
}
