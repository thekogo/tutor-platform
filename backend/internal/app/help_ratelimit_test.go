package app

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHelpRateLimit(t *testing.T) {
	h := &helpState{cfg: HelpConfig{MaxPerHour: 2}, hits: map[uuid.UUID][]time.Time{}}
	a, b := uuid.New(), uuid.New()
	t0 := time.Date(2026, 11, 22, 10, 0, 0, 0, time.UTC)

	if !h.allow(a, t0) || !h.allow(a, t0.Add(time.Minute)) {
		t.Fatal("first two messages should be allowed")
	}
	if h.allow(a, t0.Add(2*time.Minute)) {
		t.Fatal("third message within the hour should be refused")
	}
	if !h.allow(a, t0.Add(61*time.Minute)) {
		t.Fatal("budget should free up after an hour")
	}

	// Two hours later, a's timestamps have all expired: a sweep drops a.
	if !h.allow(b, t0.Add(3*time.Hour)) {
		t.Fatal("b should be allowed")
	}
	if _, ok := h.hits[a]; ok {
		t.Fatal("expired Learner should have been pruned")
	}
	if len(h.hits[b]) != 1 {
		t.Fatalf("b hits = %d, want 1", len(h.hits[b]))
	}
}
