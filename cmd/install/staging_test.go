package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStageFilesCopy(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	binaryName := "install"
	if runtime.GOOS == "windows" {
		binaryName = "install.exe"
	}
	if err := os.MkdirAll(filepath.Join(src, "core", "skills"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile manifest.json failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "core", "skills", "a.md"), []byte("a"), 0644); err != nil {
		t.Fatalf("WriteFile a.md failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, binaryName), []byte("binary"), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", binaryName, err)
	}

	err := stageFiles(src, dst, []string{"core"}, false)
	if err != nil {
		t.Fatalf("stageFiles failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "core", "skills", "a.md"))
	if err != nil {
		t.Fatalf("ReadFile a.md failed: %v", err)
	}
	if string(data) != "a" {
		t.Errorf("a.md content = %q, want 'a'", string(data))
	}
	data, err = os.ReadFile(filepath.Join(dst, binaryName))
	if err != nil {
		t.Fatalf("ReadFile %s failed: %v", binaryName, err)
	}
	if string(data) != "binary" {
		t.Errorf("%s content = %q, want 'binary'", binaryName, string(data))
	}
}

func TestStageFilesWithBusiness(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	if err := os.MkdirAll(filepath.Join(src, "core"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "business"), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile manifest.json failed: %v", err)
	}

	err := stageFiles(src, dst, []string{"core", "business"}, false)
	if err != nil {
		t.Fatalf("stageFiles failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "business")); err != nil {
		t.Fatalf("business directory was not staged: %v", err)
	}
}

func TestStageFilesSelfFileProtection(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "src") // same as src, simulating in-place install
	binaryName := "install"
	if runtime.GOOS == "windows" {
		binaryName = "install.exe"
	}
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, binaryName), []byte("binary"), 0755); err != nil {
		t.Fatalf("WriteFile %s failed: %v", binaryName, err)
	}

	err := stageFiles(src, dst, []string{"core"}, false)
	if err != nil {
		t.Fatalf("stageFiles with same src/dst failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(src, binaryName))
	if err != nil {
		t.Fatalf("ReadFile %s failed: %v", binaryName, err)
	}
	if string(data) != "binary" {
		t.Errorf("%s content = %q, want 'binary'", binaryName, string(data))
	}
}

func TestStageBrowseBinary(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	browseName := "browse"
	if runtime.GOOS == "windows" {
		browseName = "browse.exe"
	}

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, browseName), []byte("browse-binary"), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", browseName, err)
	}

	err := stageBrowseBinary(src, dst, false)
	if err != nil {
		t.Fatalf("stageBrowseBinary failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "bin", browseName))
	if err != nil {
		t.Fatalf("ReadFile %s failed: %v", browseName, err)
	}
	if string(data) != "browse-binary" {
		t.Errorf("%s content = %q, want 'browse-binary'", browseName, string(data))
	}

	// Verify executable bit is set
	info, err := os.Stat(filepath.Join(dst, "bin", browseName))
	if err != nil {
		t.Fatalf("Stat %s failed: %v", browseName, err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("%s is not executable", browseName)
	}
}

func TestStageBrowseBinaryMissing(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	err := stageBrowseBinary(src, dst, false)
	if err != nil {
		t.Fatalf("stageBrowseBinary should not fail when browse is missing: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "bin")); err == nil {
		t.Fatalf("bin directory should not be created when browse is missing")
	}
}

func TestStageBrowseBinarySymlink(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	browseName := "browse"
	if runtime.GOOS == "windows" {
		browseName = "browse.exe"
	}

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, browseName), []byte("browse-binary"), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", browseName, err)
	}

	err := stageBrowseBinary(src, dst, true)
	if err != nil {
		t.Fatalf("stageBrowseBinary symlink failed: %v", err)
	}

	info, err := os.Lstat(filepath.Join(dst, "bin", browseName))
	if err != nil {
		t.Fatalf("Lstat %s failed: %v", browseName, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("%s is not a symlink", browseName)
	}
}
