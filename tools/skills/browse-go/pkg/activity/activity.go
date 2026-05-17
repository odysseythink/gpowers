// Package activity provides real-time command streaming for the Chrome extension Side Panel.
package activity

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Entry is a single activity event.
type Entry struct {
	ID        int                    `json:"id"`
	Timestamp int64                  `json:"timestamp"`
	Type      string                 `json:"type"` // "command_start", "command_end", "navigation", "error"
	Command   string                 `json:"command,omitempty"`
	Args      []string               `json:"args,omitempty"`
	URL       string                 `json:"url,omitempty"`
	Duration  int64                  `json:"durationMs,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

const bufferCapacity = 500

type circularBuffer struct {
	entries     []Entry
	head        int
	size        int
	totalAdded  int
	mu          sync.RWMutex
}

func newCircularBuffer(capacity int) *circularBuffer {
	return &circularBuffer{entries: make([]Entry, capacity)}
}

func (b *circularBuffer) push(e Entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.totalAdded++
	e.ID = b.totalAdded
	e.Timestamp = time.Now().UnixMilli()
	b.entries[b.head] = e
	b.head = (b.head + 1) % len(b.entries)
	if b.size < len(b.entries) {
		b.size++
	}
}

func (b *circularBuffer) after(afterID int) []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if afterID >= b.totalAdded {
		return nil
	}
	var out []Entry
	// Oldest to newest
	startIdx := 0
	if b.size == len(b.entries) {
		startIdx = b.head
	}
	for i := 0; i < b.size; i++ {
		idx := (startIdx + i) % len(b.entries)
		e := b.entries[idx]
		if e.ID > afterID {
			out = append(out, e)
		}
	}
	return out
}

func (b *circularBuffer) recent(limit int) []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if limit <= 0 || limit > b.size {
		limit = b.size
	}
	out := make([]Entry, limit)
	for i := 0; i < limit; i++ {
		idx := (b.head - limit + i + len(b.entries)) % len(b.entries)
		out[i] = b.entries[idx]
	}
	return out
}

func (b *circularBuffer) stats() (total, size int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalAdded, b.size
}

var (
	buffer      = newCircularBuffer(bufferCapacity)
	subscribers sync.Map // map[*chan Entry]struct{}
)

// Emit adds an activity entry and notifies all subscribers.
func Emit(e Entry) {
	buffer.push(e)
	subscribers.Range(func(key, value interface{}) bool {
		ch := key.(*chan Entry)
		select {
		case *ch <- e:
		default:
			// Subscriber slow — drop silently (backpressure)
		}
		return true
	})
}

// EmitCommandStart records the start of a command.
func EmitCommandStart(command string, args []string, url string) {
	Emit(Entry{Type: "command_start", Command: command, Args: args, URL: url})
}

// EmitCommandEnd records the completion of a command.
func EmitCommandEnd(command string, args []string, url string, durationMs int64, err error) {
	e := Entry{Type: "command_end", Command: command, Args: args, URL: url, Duration: durationMs}
	if err != nil {
		e.Error = err.Error()
	}
	Emit(e)
}

// EmitNavigation records a page navigation.
func EmitNavigation(url string) {
	Emit(Entry{Type: "navigation", URL: url})
}

// Subscribe registers a channel to receive live activity events.
// Call Unsubscribe with the returned channel when done.
func Subscribe() *chan Entry {
	ch := make(chan Entry, 16)
	subscribers.Store(&ch, struct{}{})
	return &ch
}

// Unsubscribe removes a channel from live notifications.
func Unsubscribe(ch *chan Entry) {
	subscribers.Delete(ch)
	close(*ch)
}

// After returns entries with ID greater than afterID.
func After(afterID int) []Entry {
	return buffer.after(afterID)
}

// Recent returns the most recent N entries.
func Recent(limit int) []Entry {
	return buffer.recent(limit)
}

// Stats returns buffer statistics.
func Stats() map[string]interface{} {
	total, size := buffer.stats()
	var subscriberCount int
	subscribers.Range(func(_, _ interface{}) bool {
		subscriberCount++
		return true
	})
	return map[string]interface{}{
		"total":       total,
		"buffered":    size,
		"capacity":    bufferCapacity,
		"subscribers": subscriberCount,
	}
}

// ─── HTTP Handlers ────────────────────────────────────────

// HandleStream serves activity events via Server-Sent Events.
func HandleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Send buffered history first
	afterID := 0
	if v := r.URL.Query().Get("after"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			afterID = n
		}
	}
	for _, e := range After(afterID) {
		writeSSE(w, e)
	}

	// Subscribe to live events
	ch := Subscribe()
	defer Unsubscribe(ch)

	// Block until client disconnects or server shuts down
	done := r.Context().Done()
	for {
		select {
		case <-done:
			return
		case e, ok := <-*ch:
			if !ok {
				return
			}
			writeSSE(w, e)
		}
	}
}

func writeSSE(w http.ResponseWriter, e Entry) {
	data, _ := json.Marshal(e)
	fmt.Fprintf(w, "data: %s\n\n", data)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// HandleHistory returns recent activity entries as JSON.
func HandleHistory(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	entries := Recent(limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"stats":   Stats(),
	})
}
