package commands

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"goto https://example.com", []string{"goto", "https://example.com"}},
		{`goto "https://example.com/path with spaces"`, []string{"goto", "https://example.com/path with spaces"}},
		{"click @e1", []string{"click", "@e1"}},
		{"fill #name John Doe", []string{"fill", "#name", "John", "Doe"}},
		{``, nil},
		{`  `, nil},
		{`"quoted arg" plain`, []string{"quoted arg", "plain"}},
	}
	for _, tc := range tests {
		got := tokenize(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("tokenize(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestParseSnapshotArgs(t *testing.T) {
	tests := []struct {
		args            []string
		wantInteractive bool
		wantCompact     bool
		wantDepth       int
		wantSelector    string
		wantDiff        bool
	}{
		{[]string{"-i"}, true, false, -1, "", false},
		{[]string{"-c", "-d", "3"}, false, true, 3, "", false},
		{[]string{"-s", "#main", "--interactive"}, true, false, -1, "#main", false},
		{[]string{"--diff"}, false, false, -1, "", true},
		{[]string{"-i", "--diff"}, true, false, -1, "", true},
		{[]string{}, false, false, -1, "", false},
	}
	for _, tc := range tests {
		i, c, d, s, diff, _, _, _, _ := parseSnapshotArgsFull(tc.args)
		if i != tc.wantInteractive || c != tc.wantCompact || d != tc.wantDepth || s != tc.wantSelector || diff != tc.wantDiff {
			t.Errorf("parseSnapshotArgsFull(%v) = (%v,%v,%v,%v,%v), want (%v,%v,%v,%v,%v)",
				tc.args, i, c, d, s, diff, tc.wantInteractive, tc.wantCompact, tc.wantDepth, tc.wantSelector, tc.wantDiff)
		}
	}
}
