package main

import (
	"slices"
	"testing"
)

func TestOptionsModules(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want []string
	}{
		{"default", Options{}, []string{"core", "roles", "tools"}},
		{"core-only", Options{CoreOnly: true}, []string{"core"}},
		{"no-tools", Options{NoTools: true}, []string{"core", "roles"}},
		{"no-roles", Options{NoRoles: true}, []string{"core", "tools"}},

	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.opts.Modules()
			if !slices.Equal(got, tc.want) {
				t.Errorf("Modules() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOptionsPlatformList(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want []string
	}{
		{"multi-with-spaces", Options{Platforms: "kimi, claude-code,cursor"}, []string{"kimi", "claude-code", "cursor"}},
		{"empty", Options{Platforms: ""}, nil},
		{"only-spaces", Options{Platforms: "  "}, []string{}},
		{"empty-elements", Options{Platforms: "a,,b"}, []string{"a", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.opts.PlatformList()
			if !slices.Equal(got, tc.want) {
				t.Errorf("PlatformList() = %v, want %v", got, tc.want)
			}
		})
	}
}
