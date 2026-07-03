package models

import (
	"testing"
	"time"
)

func TestNormalizeIngressCapturedAt_staleFix(t *testing.T) {
	cap := time.Date(2026, 7, 3, 11, 28, 0, 0, time.UTC)
	recv := time.Date(2026, 7, 3, 11, 52, 0, 0, time.UTC)
	loc := &Location{CapturedAt: &cap, CreatedAt: recv}
	loc.NormalizeIngressCapturedAt()
	if !loc.EffectiveAt().Equal(recv) {
		t.Fatalf("want received time, got %v", loc.EffectiveAt())
	}
}

func TestNormalizeIngressCapturedAt_freshFix(t *testing.T) {
	cap := time.Date(2026, 7, 3, 11, 51, 30, 0, time.UTC)
	recv := time.Date(2026, 7, 3, 11, 52, 0, 0, time.UTC)
	loc := &Location{CapturedAt: &cap, CreatedAt: recv}
	loc.NormalizeIngressCapturedAt()
	if !loc.EffectiveAt().Equal(cap) {
		t.Fatalf("want captured time, got %v", loc.EffectiveAt())
	}
}
