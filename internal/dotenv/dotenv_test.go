package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadFromDefaultEnv(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", "C8X_HELLO=world\n")

	// Ensure C8X_ENV is not set so it defaults to .env
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("HELLO") != "world" {
		t.Fatalf("expected 'world', got '%s'", os.Getenv("HELLO"))
	}
}

func TestLoadFromNamedEnv(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env.staging", "C8X_STAGE=yes\n")

	t.Setenv("C8X_ENV", "staging")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("STAGE") != "yes" {
		t.Fatalf("expected 'yes', got '%s'", os.Getenv("STAGE"))
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("expected nil error when file missing, got: %v", err)
	}
}

func TestLoadFromEmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", "")

	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatalf("expected nil error for empty file, got: %v", err)
	}
}

func TestLoadFromIgnoresNonC8XVars(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", "PLAIN_KEY=should_not_load\nC8X_REAL=loaded\n")

	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("PLAIN_KEY") != "" {
		t.Fatal("non-C8X_ var should not be loaded")
	}

	if os.Getenv("REAL") != "loaded" {
		t.Fatalf("expected 'loaded', got '%s'", os.Getenv("REAL"))
	}
}

func TestLoadFromStripsQuotes(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", `C8X_QUOTED="bar"`+"\n")

	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("QUOTED") != "bar" {
		t.Fatalf("expected 'bar', got '%s'", os.Getenv("QUOTED"))
	}
}

func TestLoadFromSkipsMalformedLines(t *testing.T) {
	dir := t.TempDir()
	content := "# comment line\n\njust-a-sentence\nC8X_VALID=ok\n"
	writeEnvFile(t, dir, ".env", content)

	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("VALID") != "ok" {
		t.Fatalf("expected 'ok', got '%s'", os.Getenv("VALID"))
	}
}

func TestLoadFromMultipleVars(t *testing.T) {
	dir := t.TempDir()
	content := "C8X_A=alpha\nC8X_B=beta\nC8X_C=gamma\n"
	writeEnvFile(t, dir, ".env", content)

	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("A") != "alpha" {
		t.Fatalf("expected 'alpha', got '%s'", os.Getenv("A"))
	}
	if os.Getenv("B") != "beta" {
		t.Fatalf("expected 'beta', got '%s'", os.Getenv("B"))
	}
	if os.Getenv("C") != "gamma" {
		t.Fatalf("expected 'gamma', got '%s'", os.Getenv("C"))
	}
}

func TestLoadFromValueContainsEquals(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", "C8X_DB_URL=postgres://user:pass@host/db?sslmode=require\n")

	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("DB_URL") != "postgres://user:pass@host/db?sslmode=require" {
		t.Fatalf("expected full URL, got '%s'", os.Getenv("DB_URL"))
	}
}
