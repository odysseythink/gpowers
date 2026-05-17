package activity

import (
	"testing"
	"time"
)

func TestCircularBufferPush(t *testing.T) {
	b := newCircularBuffer(3)
	b.push(Entry{Type: "a"})
	b.push(Entry{Type: "b"})
	b.push(Entry{Type: "c"})

	total, size := b.stats()
	if total != 3 {
		t.Errorf("expected total=3, got %d", total)
	}
	if size != 3 {
		t.Errorf("expected size=3, got %d", size)
	}

	// Overflow — oldest should be evicted
	b.push(Entry{Type: "d"})
	total, size = b.stats()
	if total != 4 {
		t.Errorf("expected total=4, got %d", total)
	}
	if size != 3 {
		t.Errorf("expected size=3 after overflow, got %d", size)
	}
}

func TestCircularBufferAfter(t *testing.T) {
	b := newCircularBuffer(10)
	b.push(Entry{Type: "a"})
	b.push(Entry{Type: "b"})
	b.push(Entry{Type: "c"})

	entries := b.after(1)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries after ID 1, got %d", len(entries))
	}
	if entries[0].Type != "b" || entries[1].Type != "c" {
		t.Errorf("expected [b, c], got [%s, %s]", entries[0].Type, entries[1].Type)
	}
}

func TestCircularBufferRecent(t *testing.T) {
	b := newCircularBuffer(10)
	b.push(Entry{Type: "a"})
	b.push(Entry{Type: "b"})
	b.push(Entry{Type: "c"})

	entries := b.recent(2)
	if len(entries) != 2 {
		t.Fatalf("expected 2 recent, got %d", len(entries))
	}
	if entries[0].Type != "b" || entries[1].Type != "c" {
		t.Errorf("expected [b, c], got [%s, %s]", entries[0].Type, entries[1].Type)
	}
}

func TestEmitAndSubscribe(t *testing.T) {
	ch := Subscribe()
	defer Unsubscribe(ch)

	Emit(Entry{Type: "test"})

	select {
	case e := <-*ch:
		if e.Type != "test" {
			t.Errorf("expected type=test, got %s", e.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEmitCommandStartEnd(t *testing.T) {
	ch := Subscribe()
	defer Unsubscribe(ch)

	EmitCommandStart("text", nil, "https://example.com")
	EmitCommandEnd("text", nil, "https://example.com", 100, nil)

	var gotStart, gotEnd bool
	for i := 0; i < 2; i++ {
		select {
		case e := <-*ch:
			if e.Type == "command_start" {
				gotStart = true
			}
			if e.Type == "command_end" {
				gotEnd = true
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for events")
		}
	}
	if !gotStart {
		t.Error("expected command_start event")
	}
	if !gotEnd {
		t.Error("expected command_end event")
	}
}

func TestStats(t *testing.T) {
	b := newCircularBuffer(10)
	b.push(Entry{Type: "a"})
	b.push(Entry{Type: "b"})

	total, size := b.stats()
	if total != 2 || size != 2 {
		t.Errorf("expected total=2, size=2, got total=%d, size=%d", total, size)
	}
}
