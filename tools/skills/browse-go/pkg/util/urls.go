// Package util provides URL validation and normalization for browse navigation.
package util

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var blockedMetadataHosts = map[string]bool{
	"169.254.169.254":          true,
	"fe80::1":                  true,
	"::ffff:169.254.169.254":   true,
	"::ffff:a9fe:a9fe":         true,
	"::a9fe:a9fe":              true,
	"metadata.google.internal": true,
	"metadata.azure.internal":  true,
}

var blockedIPv6Prefixes = []string{"fc", "fd", "fe8", "fe9", "fea", "feb"}

var privateNetRegex = regexp.MustCompile(`^(10\.|172\.(1[6-9]|2[0-9]|3[01])\.|192\.168\.)`)

// isBlockedIPv6 checks if an address is in blocked IPv6 ranges.
func isBlockedIPv6(addr string) bool {
	normalized := strings.ToLower(strings.TrimPrefix(strings.TrimSuffix(addr, "]"), "["))
	if !strings.Contains(normalized, ":") {
		return false
	}
	for _, prefix := range blockedIPv6Prefixes {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	return false
}

// isMetadataIP checks if a hostname resolves to a blocked metadata IP.
func isMetadataIP(hostname string) bool {
	if blockedMetadataHosts[hostname] || isBlockedIPv6(hostname) {
		return true
	}
	// Try parsing as URL to normalize numeric forms
	if u, err := url.Parse("http://" + hostname); err == nil {
		h := strings.ToLower(u.Hostname())
		if blockedMetadataHosts[h] || isBlockedIPv6(h) {
			return true
		}
	}
	return false
}

// resolvesToBlockedIP checks DNS records for metadata IP rebinding.
func resolvesToBlockedIP(hostname string) bool {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return false
	}
	for _, addr := range addrs {
		if blockedMetadataHosts[addr] || isBlockedIPv6(addr) {
			return true
		}
	}
	return false
}

// normalizeFileURL converts non-standard file:// URLs to absolute form.
func normalizeFileURL(raw string) (string, error) {
	lower := strings.ToLower(raw)
	if !strings.HasPrefix(lower, "file:") {
		return raw, nil
	}

	// Split query/fragment
	qIdx := strings.Index(raw, "?")
	hIdx := strings.Index(raw, "#")
	delimIdx := -1
	if qIdx >= 0 && hIdx >= 0 {
		delimIdx = min(qIdx, hIdx)
	} else if qIdx >= 0 {
		delimIdx = qIdx
	} else if hIdx >= 0 {
		delimIdx = hIdx
	}

	pathPart := raw
	trailing := ""
	if delimIdx >= 0 {
		pathPart = raw[:delimIdx]
		trailing = raw[delimIdx:]
	}

	rest := pathPart[len("file:"):]

	// file:/// absolute
	if strings.HasPrefix(rest, "///") {
		if rest == "///" || rest == "////" {
			return "", fmt.Errorf("invalid file URL: empty path")
		}
		return pathPart + trailing, nil
	}

	if !strings.HasPrefix(rest, "//") {
		return "", fmt.Errorf("invalid file URL: %s", raw)
	}

	after := rest[2:]
	if after == "" {
		return "", fmt.Errorf("invalid file URL: empty path")
	}
	if after == "." || after == "./" {
		return "", fmt.Errorf("invalid file URL: would list current directory")
	}
	if after == "~" || after == "~/" {
		return "", fmt.Errorf("invalid file URL: would list home directory")
	}

	// file://~/...
	if strings.HasPrefix(after, "~/") {
		home, _ := os.UserHomeDir()
		abs := filepath.Join(home, after[2:])
		return "file://" + filepath.ToSlash(abs) + trailing, nil
	}

	// file://./...
	if strings.HasPrefix(after, "./") {
		cwd, _ := os.Getwd()
		abs := filepath.Join(cwd, after[2:])
		return "file://" + filepath.ToSlash(abs) + trailing, nil
	}

	// file://localhost/...
	if strings.HasPrefix(strings.ToLower(after), "localhost/") {
		return pathPart + trailing, nil
	}

	// Ambiguous: check if first segment looks like a host
	firstSlash := strings.Index(after, "/")
	segment := after
	if firstSlash >= 0 {
		segment = after[:firstSlash]
	}
	if strings.ContainsAny(segment, ".:%[]") {
		return "", fmt.Errorf("unsupported file URL host: %s", segment)
	}

	// Treat as cwd-relative
	cwd, _ := os.Getwd()
	abs := filepath.Join(cwd, after)
	return "file://" + filepath.ToSlash(abs) + trailing, nil
}

// ValidateNavigationURL checks and normalizes a navigation URL.
func ValidateNavigationURL(raw string) (string, error) {
	normalized, err := normalizeFileURL(raw)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(normalized)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %s", raw)
	}

	if u.Scheme == "file" {
		host := strings.ToLower(u.Hostname())
		if host != "" && host != "localhost" {
			return "", fmt.Errorf("unsupported file URL host: %s", host)
		}
		fsPath := filepath.FromSlash(u.Path)
		// Note: path-security validation would go here
		_ = fsPath
		return normalized, nil
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("blocked: scheme %q is not allowed", u.Scheme)
	}

	hostname := strings.ToLower(u.Hostname())
	if isMetadataIP(hostname) {
		return "", fmt.Errorf("blocked: %s is a cloud metadata endpoint", u.Hostname())
	}

	// Skip DNS check for loopback and private IPs
	isLoopback := hostname == "localhost" || hostname == "127.0.0.1" || hostname == "::1"
	isPrivate := privateNetRegex.MatchString(hostname)
	if !isLoopback && !isPrivate && resolvesToBlockedIP(hostname) {
		return "", fmt.Errorf("blocked: %s resolves to a cloud metadata IP", u.Hostname())
	}

	return raw, nil
}
