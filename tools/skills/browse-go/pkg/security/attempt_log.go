package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AttemptRecord is a single security event logged to disk and optionally telemetry.
type AttemptRecord struct {
	TS            string  `json:"ts"`
	URLDomain     string  `json:"urlDomain"`
	PayloadHash   string  `json:"payloadHash"`
	Confidence    float64 `json:"confidence"`
	Layer         string  `json:"layer"`
	Verdict       string  `json:"verdict"`
	GStackVersion string  `json:"gstackVersion,omitempty"`
}

const (
	maxLogBytes        = 10 * 1024 * 1024 // 10MB rotate threshold
	maxLogGenerations  = 5
)

var (
	attemptsLog = ""
	saltFile    = ""
	cachedSalt  string
)

func initAttemptPaths() {
	if attemptsLog == "" {
		attemptsLog = filepath.Join(securityDir, "attempts.jsonl")
		saltFile = filepath.Join(securityDir, "device-salt")
	}
}

// getDeviceSalt returns the per-device salt for payload hashing.
func getDeviceSalt() string {
	if cachedSalt != "" {
		return cachedSalt
	}
	data, err := os.ReadFile(saltFile)
	if err == nil && len(data) > 0 {
		cachedSalt = string(data)
		return cachedSalt
	}
	// Generate new salt
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback
		cachedSalt = fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	} else {
		cachedSalt = hex.EncodeToString(b)
	}
	_ = os.WriteFile(saltFile, []byte(cachedSalt), 0600)
	return cachedSalt
}

// HashPayload hashes a payload with the device salt.
func HashPayload(payload string) string {
	salt := getDeviceSalt()
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// rotateIfNeeded rotates the log file when it exceeds maxLogBytes.
func rotateIfNeeded() {
	info, err := os.Stat(attemptsLog)
	if err != nil || info.Size() < maxLogBytes {
		return
	}
	for i := maxLogGenerations - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", attemptsLog, i)
		dst := fmt.Sprintf("%s.%d", attemptsLog, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(attemptsLog, attemptsLog+".1")
}

// LogAttempt appends a record to the local audit log.
// Never throws — logging failure should not break the session.
func LogAttempt(record AttemptRecord) bool {
	initAttemptPaths()
	record.TS = time.Now().UTC().Format(time.RFC3339)
	_ = os.MkdirAll(securityDir, 0750)
	rotateIfNeeded()
	line, err := json.Marshal(record)
	if err != nil {
		return false
	}
	f, err := os.OpenFile(attemptsLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err == nil
}

// ExcerptForReview truncates and sanitizes text for display in the review banner.
func ExcerptForReview(text string, max int) string {
	if text == "" {
		return ""
	}
	// Strip control chars and collapse whitespace
	cleaned := controlCharRegex.ReplaceAllString(text, "")
	cleaned = whitespaceRegex.ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) <= max {
		return cleaned
	}
	return cleaned[:max] + "…"
}

var controlCharRegex = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
