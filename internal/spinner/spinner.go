package spinner

import (
	"time"
)

type Spinner struct {
	time  time.Time
	index int
	chars []string
	stop  bool
	done  string
}

func NewSpinner() *Spinner {
	chars := []string{
		"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
	}
	done := "⠿"

	return &Spinner{
		index: 0,
		time:  time.Now(),
		chars: chars,
		done:  done,
	}
}

func (s *Spinner) String() string {
	if s.stop {
		return s.done
	}

	d := time.Since(s.time)
	if d.Milliseconds() > 100 {
		s.index = (s.index + 1) % len(s.chars)
	}

	return s.chars[s.index]
}

func (s *Spinner) Stop() {
	s.stop = true
}

func (s *Spinner) Restart() {
	s.stop = false
}
