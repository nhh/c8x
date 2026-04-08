package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

// Windows-style \r\n line endings should not leak \r into values
func TestLoadFromWindowsLineEndings(t *testing.T) {
	dir := t.TempDir()
	content := "C8X_WIN=hello\r\nC8X_WIN2=world\r\n"
	writeEnvFile(t, dir, ".env", content)
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("WIN") != "hello" {
		t.Fatalf("expected 'hello', got %q (likely has \\r)", os.Getenv("WIN"))
	}

	if os.Getenv("WIN2") != "world" {
		t.Fatalf("expected 'world', got %q", os.Getenv("WIN2"))
	}
}

// A key like C8X_C8X_NESTED should become C8X_NESTED, not NESTED
// strings.Replace with -1 strips ALL occurrences of C8X_
func TestLoadFromDoubleC8XPrefix(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", "C8X_C8X_NESTED=value\n")
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	// The key should be "C8X_NESTED" (only first C8X_ stripped)
	if os.Getenv("C8X_NESTED") != "value" {
		t.Fatalf("expected key 'C8X_NESTED' with value 'value', got %q (double prefix stripped?)", os.Getenv("C8X_NESTED"))
	}
}

// Embedded quotes in JSON-like values should not be blindly stripped
func TestLoadFromEmbeddedQuotes(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", `C8X_JSON={"key":"val"}`+"\n")
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"key":"val"}`
	got := os.Getenv("JSON")
	if got != expected {
		t.Fatalf("expected %q, got %q (embedded quotes stripped?)", expected, got)
	}
}

// Empty value should work: C8X_EMPTY=
func TestLoadFromEmptyValue(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", "C8X_EMPTY=\n")
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	val, exists := os.LookupEnv("EMPTY")
	if !exists {
		t.Fatal("expected EMPTY env var to exist")
	}
	if val != "" {
		t.Fatalf("expected empty string, got %q", val)
	}
}

// Value that is just quotes: C8X_QUOTES="" should become empty
func TestLoadFromQuotedEmpty(t *testing.T) {
	dir := t.TempDir()
	writeEnvFile(t, dir, ".env", `C8X_QE=""`+"\n")
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("QE") != "" {
		t.Fatalf("expected empty string, got %q", os.Getenv("QE"))
	}
}

// Leading spaces in values should be preserved
func TestLoadFromValueWithSpaces(t *testing.T) {
	dir := t.TempDir()
	// Second line ensures SP's trailing space isn't eaten by TrimSpace on EOF
	writeEnvFile(t, dir, ".env", "C8X_SP= hello world \nC8X_AFTER=ok\n")
	t.Setenv("C8X_ENV", "")

	err := LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("SP") != " hello world " {
		t.Fatalf("expected ' hello world ', got %q", os.Getenv("SP"))
	}
}

// Symlink to .env file - ensure we don't break on symlinks
func TestLoadFromSymlink(t *testing.T) {
	dir := t.TempDir()
	realDir := t.TempDir()

	writeEnvFile(t, realDir, ".env", "C8X_LINKED=yes\n")

	// Create symlink from dir/.env -> realDir/.env
	err := os.Symlink(
		filepath.Join(realDir, ".env"),
		filepath.Join(dir, ".env"),
	)
	if err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	t.Setenv("C8X_ENV", "")

	err = LoadFrom(dir)
	if err != nil {
		t.Fatal(err)
	}

	if os.Getenv("LINKED") != "yes" {
		t.Fatalf("expected 'yes', got %q", os.Getenv("LINKED"))
	}
}
