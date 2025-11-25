package testhelpers

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"testing"
)

var dotEnvOnce sync.Once

// LoadDotEnv reads workspace .env and applies any missing variables.
// It's safe to call multiple times because it is guarded by sync.Once.
func LoadDotEnv(t *testing.T) {
	t.Helper()
	dotEnvOnce.Do(func() {
		file, err := os.Open(".env")
		if err != nil {
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "export ") {
				line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "'\"")
			if key == "" {
				continue
			}
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}

		if err := scanner.Err(); err != nil {
			t.Logf("warning: failed to parse .env: %v", err)
		}
	})
}
