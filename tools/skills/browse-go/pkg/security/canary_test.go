package security

import (
	"testing"
)

func TestGenerateCanary(t *testing.T) {
	c1 := GenerateCanary()
	c2 := GenerateCanary()
	if c1 == "" {
		t.Errorf("canary should not be empty")
	}
	if c1 == c2 {
		t.Errorf("canaries should be unique: %q == %q", c1, c2)
	}
	if len(c1) < 10 {
		t.Errorf("canary too short: %q", c1)
	}
}

func TestInjectCanary(t *testing.T) {
	prompt := "You are a helpful assistant."
	canary := "CANARY-ABC123"
	injected := InjectCanary(prompt, canary)
	if !contains(injected, canary) {
		t.Errorf("injected prompt missing canary")
	}
	if !contains(injected, "SECURITY CANARY") {
		t.Errorf("injected prompt missing instruction")
	}
}

func TestCheckCanaryInString(t *testing.T) {
	canary := "CANARY-TEST123"
	if !CheckCanaryInString("hello CANARY-TEST123 world", canary) {
		t.Errorf("should detect canary in string")
	}
	if CheckCanaryInString("hello world", canary) {
		t.Errorf("should not detect canary when absent")
	}
}

func TestCheckCanaryInStructure(t *testing.T) {
	canary := "CANARY-STRUCT"
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"string hit", "hello " + canary + " world", true},
		{"string miss", "hello world", false},
		{"array hit", []interface{}{"a", "b", canary}, true},
		{"array miss", []interface{}{"a", "b", "c"}, false},
		{"map hit", map[string]interface{}{"key": canary}, true},
		{"map miss", map[string]interface{}{"key": "value"}, false},
		{"nested hit", map[string]interface{}{"nested": map[string]interface{}{"deep": canary}}, true},
		{"nil", nil, false},
		{"number", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckCanaryInStructure(tt.value, canary)
			if got != tt.want {
				t.Errorf("CheckCanaryInStructure(%v, %q) = %v, want %v", tt.value, canary, got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
