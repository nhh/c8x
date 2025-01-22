package dotenv

import (
	"fmt"
	"os"
	"path"
	"strings"
)

func Load() error {
	envToLoad := os.Getenv("K8X_ENV")

	if envToLoad == "" {
		// Defaults to this file
		envToLoad = ".env"
	} else {
		// Making .env.production out of K8X_ENV=production
		envToLoad = ".env." + envToLoad
	}

	cwd, err := os.Getwd()

	if err != nil {
		return fmt.Errorf("getting current working directory: %w", err)
	}

	_, err = os.Stat(path.Join(cwd, envToLoad))

	if err != nil {
		return nil
	}

	file, err := os.ReadFile(envToLoad)

	if err != nil {
		return fmt.Errorf("reading dot env file: %s: %w", envToLoad, err)
	}

	// Split by newlines
	lines := strings.Split(strings.TrimSpace(string(file)), "\n")

	if len(lines) == 0 {
		// The file is empty :/
		return nil
	}

	for _, line := range lines {

		pair := strings.Split(line, "=")

		if len(pair) != 2 {
			// Ignore empty lines, comments and single sentences that dont have = innit
			continue
		}

		key := pair[0]

		// Make KEY="VALUE" to KEY=VALUE
		value := strings.Replace(pair[1], "\"", "", -1)

		if !strings.HasPrefix(key, "K8X_") {
			// Skip all variables that dont start with K8X_
			continue
		}

		key = strings.Replace(key, "K8X_", "", -1)

		err := os.Setenv(key, value)

		if err != nil {
			return fmt.Errorf("setting env var %s with value %s: %w", key, value, err)
		}
	}

	return nil
}
