package dotenv

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	if os.Getenv("KEY") != "" {
		t.Fatalf(`Initial expectation is not met, KEY alread in os.Getenv`)
	}

	// Configuring to use top level .env.test file
	err := os.Setenv("K8X_ENV", "test")

	if err != nil {
		t.Fatalf(err.Error())
	}

	_ = Load()

	if os.Getenv("KEY") != "" {
		t.Fatalf(`KEY is not empty! Loading env variables without K8X prefix`)
	}

	if os.Getenv("HEY") != "" {
		t.Fatalf(`HEY is not empty! Loading env variables without K8X prefix`)
	}

	if os.Getenv("TEST") != "TEST" {
		t.Fatalf(`TEST doesnt have TEST as value!: %s`, os.Getenv("TEST"))
	}

	if os.Getenv("TEST2") != `abcde` {
		t.Fatalf(`TEST2 is not: %s`, `abcde`)
	}

}
