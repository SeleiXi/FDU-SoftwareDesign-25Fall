package statistics_test

import (
	"testing"
	"time"

	"softwaredesign/src/statistics"
)

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time { return f.now }

func (f *fakeClock) Advance(d time.Duration) { f.now = f.now.Add(d) }

func TestTrackerSwitchAndDuration(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	tracker := statistics.NewTracker()
	tracker.WithClock(clock)

	tracker.Switch("", "a.txt")
	clock.Advance(30 * time.Second)
	tracker.Switch("a.txt", "b.txt")
	clock.Advance(90 * time.Second)
	tracker.Switch("b.txt", "")

	if got := statistics.FormatDuration(tracker.Duration("a.txt")); got != "30秒" {
		t.Fatalf("unexpected duration for a.txt: %s", got)
	}
	if got := statistics.FormatDuration(tracker.Duration("b.txt")); got != "1分钟" {
		t.Fatalf("unexpected duration for b.txt: %s", got)
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                            "0秒",
		45 * time.Second:             "45秒",
		5 * time.Minute:              "5分钟",
		2*time.Hour + 15*time.Minute: "2小时15分钟",
		24 * time.Hour:               "1天",
		25 * time.Hour:               "1天1小时",
	}
	for input, expected := range cases {
		if got := statistics.FormatDuration(input); got != expected {
			t.Fatalf("format mismatch for %v: want %s, got %s", input, expected, got)
		}
	}
}
