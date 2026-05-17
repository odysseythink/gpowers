package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// CapturedResponse holds a single intercepted network response.
type CapturedResponse struct {
	URL           string            `json:"url"`
	Status        int               `json:"status"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body"`
	ContentType   string            `json:"contentType"`
	Timestamp     int64             `json:"timestamp"`
	Size          int               `json:"size"`
	BodyTruncated bool              `json:"bodyTruncated"`
}

const (
	defaultMaxBufferSize = 50 * 1024 * 1024 // 50 MB total
	defaultMaxEntrySize  = 5 * 1024 * 1024  // 5 MB per response body
)

// SizeCappedBuffer stores captured responses with a total memory cap.
// When the cap is exceeded, oldest entries are evicted.
type SizeCappedBuffer struct {
	mu        sync.RWMutex
	entries   []CapturedResponse
	totalSize int
	maxSize   int
}

// NewSizeCappedBuffer creates a buffer with the given cap (default 50 MB).
func NewSizeCappedBuffer(maxSize int) *SizeCappedBuffer {
	if maxSize <= 0 {
		maxSize = defaultMaxBufferSize
	}
	return &SizeCappedBuffer{maxSize: maxSize}
}

// Push adds a response to the buffer, evicting oldest entries if needed.
func (b *SizeCappedBuffer) Push(entry CapturedResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.totalSize+entry.Size > b.maxSize && len(b.entries) > 0 {
		b.totalSize -= b.entries[0].Size
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
	b.totalSize += entry.Size
}

// ToArray returns a shallow copy of all entries.
func (b *SizeCappedBuffer) ToArray() []CapturedResponse {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]CapturedResponse, len(b.entries))
	copy(out, b.entries)
	return out
}

// Len returns the number of stored entries.
func (b *SizeCappedBuffer) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.entries)
}

// ByteSize returns the total byte size of stored bodies.
func (b *SizeCappedBuffer) ByteSize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalSize
}

// Clear empties the buffer.
func (b *SizeCappedBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = nil
	b.totalSize = 0
}

// ExportToFile writes all entries as JSONL (one JSON object per line).
func (b *SizeCappedBuffer) ExportToFile(filePath string) (int, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	f, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, e := range b.entries {
		if err := enc.Encode(e); err != nil {
			return 0, err
		}
	}
	return len(b.entries), nil
}

// Summary returns a human-readable summary of captured responses.
func (b *SizeCappedBuffer) Summary() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.entries) == 0 {
		return "No captured responses."
	}
	lines := make([]string, len(b.entries))
	for i, e := range b.entries {
		url := e.URL
		if len(url) > 100 {
			url = url[:100] + "..."
		}
		trunc := ""
		if e.BodyTruncated {
			trunc = ", truncated"
		}
		lines[i] = fmt.Sprintf("  [%d] %d %s (%dKB%s)", i+1, e.Status, url, e.Size/1024, trunc)
	}
	return fmt.Sprintf("%d responses (%dKB total):\n%s",
		len(b.entries), b.totalSize/1024, joinLines(lines))
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
