package browser

import (
	"os"
	"strings"
	"testing"
)

func TestSizeCappedBufferPushAndEvict(t *testing.T) {
	// Cap of 100 bytes
	b := NewSizeCappedBuffer(100)

	b.Push(CapturedResponse{URL: "http://a", Size: 40})
	b.Push(CapturedResponse{URL: "http://b", Size: 40})
	b.Push(CapturedResponse{URL: "http://c", Size: 40})

	// Total = 120 > 100, so oldest (a) should be evicted
	if b.Len() != 2 {
		t.Fatalf("expected 2 entries after eviction, got %d", b.Len())
	}
	arr := b.ToArray()
	if arr[0].URL != "http://b" || arr[1].URL != "http://c" {
		t.Fatalf("unexpected entries: %+v", arr)
	}
	if b.ByteSize() != 80 {
		t.Fatalf("expected byteSize 80, got %d", b.ByteSize())
	}
}

func TestSizeCappedBufferClear(t *testing.T) {
	b := NewSizeCappedBuffer(100)
	b.Push(CapturedResponse{URL: "http://x", Size: 10})
	b.Clear()
	if b.Len() != 0 || b.ByteSize() != 0 {
		t.Fatal("expected empty after clear")
	}
}

func TestSizeCappedBufferExport(t *testing.T) {
	b := NewSizeCappedBuffer(100)
	b.Push(CapturedResponse{URL: "http://a", Status: 200, Size: 5, Body: "hello"})
	b.Push(CapturedResponse{URL: "http://b", Status: 404, Size: 5, Body: "nope"})

	tmp := t.TempDir() + "/out.jsonl"
	n, err := b.ExportToFile(tmp)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 lines exported, got %d", n)
	}

	data, err := os.ReadFile(tmp)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], `"url":"http://a"`) {
		t.Fatalf("missing URL in export: %s", lines[0])
	}
}

func TestSizeCappedBufferSummary(t *testing.T) {
	b := NewSizeCappedBuffer(100)
	if b.Summary() != "No captured responses." {
		t.Fatalf("unexpected empty summary: %s", b.Summary())
	}

	b.Push(CapturedResponse{URL: "http://example.com/path", Status: 200, Size: 2048})
	summary := b.Summary()
	if !strings.Contains(summary, "1 responses") {
		t.Fatalf("expected '1 responses' in summary: %s", summary)
	}
	if !strings.Contains(summary, "200") {
		t.Fatalf("expected status in summary: %s", summary)
	}
}
