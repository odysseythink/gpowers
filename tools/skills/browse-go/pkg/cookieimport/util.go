package cookieimport

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// minimalEnv returns a minimal environment for subprocess calls.
func minimalEnv() []string {
	keep := []string{"PATH", "HOME", "USER", "LANG", "LC_ALL", "TERM"}
	var out []string
	for _, k := range keep {
		if v := os.Getenv(k); v != "" {
			out = append(out, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return out
}
