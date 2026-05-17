package buffers

import (
	"testing"
)

func TestCircular(t *testing.T) {
	c := NewCircular[Entry](3)
	if c.Len() != 0 {
		t.Fatalf("expected len 0, got %d", c.Len())
	}

	c.Add(Entry{Timestamp: 1, Message: "a"})
	c.Add(Entry{Timestamp: 2, Message: "b"})
	c.Add(Entry{Timestamp: 3, Message: "c"})

	all := c.All()
	if len(all) != 3 || all[0].Message != "a" || all[2].Message != "c" {
		t.Fatalf("unexpected order: %+v", all)
	}

	c.Add(Entry{Timestamp: 4, Message: "d"})
	all = c.All()
	if len(all) != 3 || all[0].Message != "b" || all[2].Message != "d" {
		t.Fatalf("unexpected wrap order: %+v", all)
	}
}

func TestCircularClear(t *testing.T) {
	c := NewCircular[string](5)
	c.Add("a")
	c.Add("b")
	if c.Len() != 2 {
		t.Fatalf("expected len 2, got %d", c.Len())
	}
	c.Clear()
	if c.Len() != 0 {
		t.Fatalf("expected len 0 after clear, got %d", c.Len())
	}
	if len(c.All()) != 0 {
		t.Fatalf("expected empty All() after clear")
	}
	// Should be usable after clear
	c.Add("c")
	all := c.All()
	if len(all) != 1 || all[0] != "c" {
		t.Fatalf("unexpected after re-add: %v", all)
	}
}
