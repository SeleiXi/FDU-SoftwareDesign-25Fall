package statistics

import (
	"fmt"
	"sync"
	"time"
)

// Clock abstracts time retrieval for testing.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Tracker records per-file editing durations within a session.
type Tracker struct {
	mu        sync.Mutex
	durations map[string]time.Duration
	active    string
	started   time.Time
	clock     Clock
}

// NewTracker constructs a tracker with a real clock.
func NewTracker() *Tracker {
	return &Tracker{
		durations: map[string]time.Duration{},
		clock:     realClock{},
	}
}

// WithClock swaps the underlying clock (primarily for tests).
func (t *Tracker) WithClock(clock Clock) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if clock == nil {
		t.clock = realClock{}
		return
	}
	t.clock = clock
}

// Switch transitions timing from prev to next active file.
func (t *Tracker) Switch(prev, next string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.clock.Now()
	if prev != "" && t.active == prev {
		t.durations[prev] += now.Sub(t.started)
	}
	if next != "" {
		if _, ok := t.durations[next]; !ok {
			t.durations[next] = 0
		}
		t.active = next
		t.started = now
	} else {
		t.active = ""
	}
}

// Close stops tracking for the provided file and clears its history.
func (t *Tracker) Close(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.clock.Now()
	if path == "" {
		return
	}
	if t.active == path {
		t.durations[path] += now.Sub(t.started)
		t.active = ""
	}
	delete(t.durations, path)
}

// StopAll flushes the active timer without clearing durations.
func (t *Tracker) StopAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.active == "" {
		return
	}
	t.durations[t.active] += t.clock.Now().Sub(t.started)
	t.active = ""
}

// Duration reports the accumulated duration for the file.
func (t *Tracker) Duration(path string) time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	total := t.durations[path]
	if path != "" && path == t.active {
		total += t.clock.Now().Sub(t.started)
	}
	return total
}

// FormatDuration renders a duration following the lab specification.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return "0秒"
	}
	seconds := int(d / time.Second)
	if seconds < 60 {
		return fmt.Sprintf("%d秒", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%d分钟", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		remMinutes := minutes % 60
		if remMinutes == 0 {
			return fmt.Sprintf("%d小时", hours)
		}
		return fmt.Sprintf("%d小时%d分钟", hours, remMinutes)
	}
	days := hours / 24
	remHours := hours % 24
	if remHours == 0 {
		return fmt.Sprintf("%d天", days)
	}
	return fmt.Sprintf("%d天%d小时", days, remHours)
}
