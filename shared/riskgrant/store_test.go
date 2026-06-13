package riskgrant

import (
	"testing"
	"time"
)

func TestRiskGrantValidForSameScopeWithinTTL(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	store := NewStoreWithClock(5*time.Minute, func() time.Time { return now })
	store.Grant("line", "傳送", "line-user-1", "high")

	if !store.HasValid("line", "傳送", "line-user-1", "high") {
		t.Fatal("expected same scope to be valid")
	}
	if store.HasValid("line", "傳送", "line-user-2", "high") {
		t.Fatal("different target must require confirmation")
	}
}

func TestRiskGrantExpiresAfterTTL(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	store := NewStoreWithClock(5*time.Minute, func() time.Time { return now })
	store.Grant("line", "傳送", "line-user-1", "high")

	now = now.Add(5*time.Minute + time.Second)
	if store.HasValid("line", "傳送", "line-user-1", "high") {
		t.Fatal("grant should expire after ttl")
	}
}
