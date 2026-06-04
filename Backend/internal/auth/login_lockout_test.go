package auth

import (
	"testing"
	"time"
)

func TestLoginLockoutEscalation(t *testing.T) {
	store := newLoginLockoutStore()
	email := "user@example.com"

	for i := 0; i < loginMaxFailuresBeforeLock-1; i++ {
		blocked, msg := store.recordFailure(email)
		if blocked || msg != "" {
			t.Fatalf("attempt %d should not lock yet", i+1)
		}
	}

	blocked, msg := store.recordFailure(email)
	if !blocked || msg == "" {
		t.Fatal("5th failure should lock account")
	}

	blocked, _ = store.check(email)
	if !blocked {
		t.Fatal("account should remain locked immediately after threshold")
	}

	rec := store.recs[email]
	if rec.LockoutLevel != 1 {
		t.Fatalf("lockout level = %d, want 1", rec.LockoutLevel)
	}
	if got := loginLockoutDuration(0); got != 10*time.Minute {
		t.Fatalf("first lock duration = %v, want 10m", got)
	}

	store.recordSuccess(email)
	if _, ok := store.recs[email]; ok {
		t.Fatal("success should clear lockout state")
	}
}

func TestLoginLockoutSecondTier(t *testing.T) {
	store := newLoginLockoutStore()
	email := "user@example.com"

	lockOnce := func() time.Duration {
		for i := 0; i < loginMaxFailuresBeforeLock; i++ {
			store.recordFailure(email)
		}
		rec := store.recs[email]
		wait := rec.LockedUntil.Sub(time.Now())
		rec.LockedUntil = time.Now().Add(-time.Second)
		return wait
	}

	first := lockOnce()
	if first < 9*time.Minute || first > 11*time.Minute {
		t.Fatalf("first lock wait = %v, want ~10m", first)
	}

	second := lockOnce()
	if second < 59*time.Minute || second > 61*time.Minute {
		t.Fatalf("second lock wait = %v, want ~1h", second)
	}
}

func TestLoginLockoutWhileLockedDoesNotEscalate(t *testing.T) {
	store := newLoginLockoutStore()
	email := "user@example.com"
	for i := 0; i < loginMaxFailuresBeforeLock; i++ {
		store.recordFailure(email)
	}
	rec := store.recs[email]
	level := rec.LockoutLevel

	blocked, _ := store.recordFailure(email)
	if !blocked {
		t.Fatal("extra failure while locked should stay blocked")
	}
	if rec.LockoutLevel != level {
		t.Fatalf("lockout level changed while locked: %d -> %d", level, rec.LockoutLevel)
	}
}

func TestNormalizeLoginEmail(t *testing.T) {
	if got := NormalizeLoginEmail("  User@Example.COM "); got != "user@example.com" {
		t.Fatalf("normalize = %q", got)
	}
}
