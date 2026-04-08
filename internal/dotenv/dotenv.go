package dotenv

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func Load() error {
	cwd, err := os.Getwd()

	if err != nil {
		return fmt.Errorf("getting current working directory: %w", err)
	}

	return LoadFrom(cwd)
}

func LoadFrom(dir string) error {
	envToLoad := os.Getenv("C8X_ENV")

	if envToLoad == "" {
		// Defaults to this file
		envToLoad = ".env"
	} else {
		// Making .env.production out of C8X_ENV=production
		envToLoad = ".env." + envToLoad
	}

	filePath := path.Join(dir, envToLoad)

	_, err := os.Stat(filePath)

	if err != nil {
		return nil
	}

	file, err := os.ReadFile(filePath)

	if err != nil {
		return fmt.Errorf("reading dot env file: %s: %w", envToLoad, err)
	}

	// Normalize \r\n to \n for Windows compatibility
	content := strings.ReplaceAll(strings.TrimSpace(string(file)), "\r\n", "\n")

	lines := strings.Split(content, "\n")

	if len(lines) == 0 {
		// The file is empty :/
		return nil
	}

	for _, line := range lines {

		pair := strings.SplitN(line, "=", 2)

		if len(pair) != 2 {
			// Ignore empty lines, comments and single sentences that dont have = innit
			continue
		}

		key := pair[0]
		value := pair[1]

		// Strip surrounding quotes only (not embedded ones)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		if !strings.HasPrefix(key, "C8X_") {
			// Skip all variables that dont start with C8X_
			continue
		}

		key = strings.TrimPrefix(key, "C8X_")

		err := os.Setenv(key, value)

		if err != nil {
			return fmt.Errorf("setting env var %s with value %s: %w", key, value, err)
		}
	}

	return nil
}
