package schedule

import (
	"context"
	"testing"
	"time"
)

func TestParseCron(t *testing.T) {
	c, err := parseCron("*/5 9-17 * * 1-5")
	if err != nil {
		t.Fatal(err)
	}
	tm := time.Date(2026, 8, 20, 10, 10, 0, 0, time.UTC) // Thursday
	if !c.matches(tm) {
		t.Fatal("expected match")
	}
	tm = time.Date(2026, 8, 20, 10, 11, 0, 0, time.UTC)
	if c.matches(tm) {
		t.Fatal("11 should not match */5")
	}
}

func TestFluentName(t *testing.T) {
	s := New()
	s.Every(time.Hour, func(context.Context) {}).Name("hourly").WithoutOverlapping()
	s.Cron("0 * * * *", func(context.Context) {})
}
