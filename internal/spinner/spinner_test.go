package spinner

import (
	"testing"
	"time"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner()

	if s.index != 0 {
		t.Fatalf("expected index 0, got %d", s.index)
	}

	if s.stop {
		t.Fatal("expected stop to be false")
	}

	if len(s.chars) != 10 {
		t.Fatalf("expected 10 spinner chars, got %d", len(s.chars))
	}

	if s.done != "⠿" {
		t.Fatalf("expected done char ⠿, got %s", s.done)
	}
}

func TestSpinnerStop(t *testing.T) {
	s := NewSpinner()
	s.Stop()

	if s.String() != "⠿" {
		t.Fatalf("expected done char after Stop(), got %s", s.String())
	}
}

func TestSpinnerRestart(t *testing.T) {
	s := NewSpinner()
	s.Stop()
	s.Restart()

	result := s.String()
	if result == s.done {
		t.Fatal("expected frame char after Restart(), got done char")
	}
}

func TestSpinnerStringCycles(t *testing.T) {
	s := NewSpinner()
	// Set time to the past so the 100ms threshold is exceeded
	s.time = time.Now().Add(-200 * time.Millisecond)

	first := s.String()
	// After calling String() with elapsed > 100ms, index should have advanced
	if s.index != 1 {
		t.Fatalf("expected index 1 after first call, got %d", s.index)
	}

	// Reset time again to trigger another advance
	s.time = time.Now().Add(-200 * time.Millisecond)
	second := s.String()

	if first == second {
		t.Fatal("expected different frames after cycling")
	}

	if s.index != 2 {
		t.Fatalf("expected index 2 after second call, got %d", s.index)
	}
}

func TestSpinnerStringWrapsAround(t *testing.T) {
	s := NewSpinner()
	s.index = len(s.chars) - 1
	s.time = time.Now().Add(-200 * time.Millisecond)

	s.String()

	if s.index != 0 {
		t.Fatalf("expected index to wrap to 0, got %d", s.index)
	}
}
