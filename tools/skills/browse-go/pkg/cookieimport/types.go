package cookieimport

import (
	"github.com/chromedp/cdproto/network"
)

// BrowserInfo describes a supported Chromium-based browser.
type BrowserInfo struct {
	Name            string
	DataDir         string // macOS data dir relative to Application Support
	KeychainService string // macOS keychain service name
	Aliases         []string
	LinuxDataDir    string // Linux data dir relative to ~/.config
	LinuxApp        string // Linux application name for secret-tool
	WindowsDataDir  string // Windows data dir relative to %LOCALAPPDATA%
}

// ProfileEntry is a browser profile.
type ProfileEntry struct {
	Name        string
	DisplayName string
}

// DomainEntry is a cookie domain with count.
type DomainEntry struct {
	Domain string
	Count  int
}

// CDPCookie is a cookie in CDP-compatible format.
type CDPCookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  float64
	Secure   bool
	HTTPOnly bool
	SameSite network.CookieSameSite
}

// ImportResult is the result of importing cookies.
type ImportResult struct {
	Cookies      []*CDPCookie
	Count        int
	Failed       int
	DomainCounts map[string]int
}

// Error is a cookie import error with a code.
type Error struct {
	Code   string
	Action string // "retry" or empty
	Msg    string
}

func (e *Error) Error() string { return e.Msg }

// NewError creates a new cookie import error.
func NewError(code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

// NewRetryError creates a retryable cookie import error.
func NewRetryError(code, msg string) *Error {
	return &Error{Code: code, Action: "retry", Msg: msg}
}

// browserPlatform identifies the host OS.
type browserPlatform string

const (
	platformDarwin browserPlatform = "darwin"
	platformLinux  browserPlatform = "linux"
	platformWin32  browserPlatform = "win32"
)

// browserMatch holds a resolved browser + platform + DB path.
type browserMatch struct {
	browser  *BrowserInfo
	platform browserPlatform
	dbPath   string
}

// rawCookieRow is a row from the cookies table.
type rawCookieRow struct {
	HostKey        string
	Name           string
	Value          string
	EncryptedValue []byte
	Path           string
	ExpiresUTC     int64
	IsSecure       int
	IsHTTPOnly     int
	HasExpires     int
	SameSite       int
}
