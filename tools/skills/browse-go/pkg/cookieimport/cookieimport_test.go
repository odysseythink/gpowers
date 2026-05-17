package cookieimport

import (
	"testing"
)

func TestResolveBrowser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"chrome lowercase", "chrome", "Chrome", false},
		{"Chrome capitalized", "Chrome", "Chrome", false},
		{"chromium alias", "chromium", "Chromium", false},
		{"brave alias", "brave", "Brave", false},
		{"edge alias", "edge", "Edge", false},
		{"comet alias", "comet", "Comet", false},
		{"perplexity alias", "perplexity", "Comet", false},
		{"unknown browser", "safari", "", true},
		{"empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBrowser(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.want {
				t.Errorf("resolveBrowser(%q) = %q, want %q", tt.input, got.Name, tt.want)
			}
		})
	}
}

func TestValidateProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		wantErr bool
	}{
		{"Default", "Default", false},
		{"Profile 1", "Profile 1", false},
		{"with slash", "foo/bar", true},
		{"with dotdot", "foo..bar", true},
		{"with backslash", `foo\bar`, true},
		{"null byte", "foo\x00bar", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfile(tt.profile)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error for profile %q", tt.profile)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestChromiumEpochToUnix(t *testing.T) {
	// 2024-01-01 00:00:00 UTC in Chromium epoch
	// Unix seconds for 2024-01-01 = 1704067200
	// Chromium epoch = 1704067200 * 1e6 + 11644473600000000 = 13348540800000000
	const jan2024Chromium = 13348540800000000
	got := chromiumEpochToUnix(jan2024Chromium, 1)
	if got != 1704067200 {
		t.Errorf("chromiumEpochToUnix(2024-01-01) = %v, want 1704067200", got)
	}

	// Session cookie
	got2 := chromiumEpochToUnix(0, 0)
	if got2 != -1 {
		t.Errorf("chromiumEpochToUnix(session) = %v, want -1", got2)
	}
}

func TestMapSameSite(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "None"},
		{1, "Lax"},
		{2, "Strict"},
		{99, "Lax"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := mapSameSite(tt.input)
			if string(got) != tt.want {
				t.Errorf("mapSameSite(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPKCS7Unpad(t *testing.T) {
	tests := []struct {
		input []byte
		want  []byte
	}{
		{[]byte("hello world\x05\x05\x05\x05\x05"), []byte("hello world")},
		{[]byte("abc\x01"), []byte("abc")},
		{[]byte{}, []byte{}},
		{[]byte{0x01}, []byte{}}, // valid 1-byte padding, returns empty
	}
	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			got := pkcs7Unpad(tt.input)
			if string(got) != string(tt.want) {
				t.Errorf("pkcs7Unpad(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeriveKey(t *testing.T) {
	// Linux v10 key: "peanuts", iter=1
	key, err := deriveKey("peanuts", 1)
	if err != nil {
		t.Fatalf("deriveKey failed: %v", err)
	}
	if len(key) != 16 {
		t.Errorf("deriveKey length = %d, want 16", len(key))
	}

	// macOS v10 key: some password, iter=1003
	key2, err := deriveKey("testpassword", 1003)
	if err != nil {
		t.Fatalf("deriveKey failed: %v", err)
	}
	if len(key2) != 16 {
		t.Errorf("deriveKey length = %d, want 16", len(key2))
	}
}

func TestJSONExtractString(t *testing.T) {
	tests := []struct {
		name string
		data string
		keys []string
		want string
	}{
		{
			name: "simple nested",
			data: `{"profile":{"name":"Test Profile"}}`,
			keys: []string{`"profile"`, `"name"`},
			want: "Test Profile",
		},
		{
			name: "account email",
			data: `{"account_info":[{"email":"test@example.com"}]}`,
			keys: []string{`"account_info"`, `"email"`},
			want: "test@example.com",
		},
		{
			name: "missing key",
			data: `{"other":{}}`,
			keys: []string{`"profile"`, `"name"`},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonExtractString([]byte(tt.data), tt.keys...)
			if got != tt.want {
				t.Errorf("jsonExtractString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListSupportedBrowserNames(t *testing.T) {
	names := ListSupportedBrowserNames()
	if len(names) == 0 {
		t.Fatal("ListSupportedBrowserNames returned empty")
	}
	foundChrome := false
	for _, n := range names {
		if n == "Chrome" {
			foundChrome = true
			break
		}
	}
	if !foundChrome {
		t.Error("Chrome not in supported browser names")
	}
}

func TestFindInstalledBrowsers(t *testing.T) {
	// This may return empty on CI machines without Chromium browsers installed.
	// We just verify it doesn't panic.
	browsers := FindInstalledBrowsers()
	t.Logf("Found %d installed browsers", len(browsers))
}

func TestToCDPCookie(t *testing.T) {
	row := &rawCookieRow{
		HostKey:    ".example.com",
		Name:       "session_id",
		Value:      "abc123",
		Path:       "/",
		ExpiresUTC: 13348540800000000, // 2024-01-01
		IsSecure:   1,
		IsHTTPOnly: 1,
		HasExpires: 1,
		SameSite:   2,
	}
	c := toCDPCookie(row, "abc123")
	if c.Name != "session_id" {
		t.Errorf("Name = %q, want session_id", c.Name)
	}
	if c.Value != "abc123" {
		t.Errorf("Value = %q, want abc123", c.Value)
	}
	if c.Domain != ".example.com" {
		t.Errorf("Domain = %q, want .example.com", c.Domain)
	}
	if !c.Secure {
		t.Error("Secure should be true")
	}
	if !c.HTTPOnly {
		t.Error("HTTPOnly should be true")
	}
	if c.SameSite != "Strict" {
		t.Errorf("SameSite = %q, want Strict", c.SameSite)
	}
	if c.Expires != 1704067200 {
		t.Errorf("Expires = %v, want 1704067200", c.Expires)
	}
}

func TestBase64RoundTrip(t *testing.T) {
	data := []byte("hello world! 你好世界")
	encoded := base64Encode(data)
	decoded, err := base64Decode(encoded)
	if err != nil {
		t.Fatalf("base64Decode failed: %v", err)
	}
	if string(decoded) != string(data) {
		t.Errorf("round-trip failed: %q != %q", decoded, data)
	}
}
