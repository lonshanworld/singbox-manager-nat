package auth

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnv reads a basic .env file and sets values into the environment
func LoadEnv(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // If file doesn't exist, we just rely on existing env vars if any
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			
			// Remove surrounding quotes if present
			if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) || 
			   (strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
				val = val[1 : len(val)-1]
			}

			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}
