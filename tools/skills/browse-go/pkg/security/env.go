package security

import "os"

// GetEnv returns the value of the named environment variable, or defaultValue if not set.
func GetEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
