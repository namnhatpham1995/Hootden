package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestAttemptLimiter_OpensRefusesAtThresholdThenLifts(t *testing.T) {
	l := NewAttemptLimiter(2, 20*time.Millisecond, 10)

	if !l.allow("k") {
		t.Fatal("first attempt should open the window and be allowed")
	}
	if !l.allow("k") {
		t.Fatal("second attempt is still within the limit")
	}
	if l.allow("k") {
		t.Fatal("third attempt should be refused at the threshold")
	}

	time.Sleep(30 * time.Millisecond)

	if !l.allow("k") {
		t.Fatal("the window should have lifted on its own after it elapsed")
	}
}

func TestAttemptLimiter_AllowAttempt_IndependentKeys(t *testing.T) {
	l := NewAttemptLimiter(1, time.Hour, 10)

	if !l.AllowAttempt("login", "a@example.com", "1.2.3.4") {
		t.Fatal("first attempt should be allowed")
	}
	if l.AllowAttempt("login", "b@example.com", "1.2.3.4") {
		t.Fatal("expected refusal: the address key is already at its limit")
	}
	if l.AllowAttempt("login", "a@example.com", "9.9.9.9") {
		t.Fatal("expected refusal: the email key is already at its limit")
	}
	if !l.AllowAttempt("login", "c@example.com", "8.8.8.8") {
		t.Fatal("a fresh email and a fresh address should still be allowed")
	}
}

func TestAttemptLimiter_AllowAttempt_ScopedByEndpoint(t *testing.T) {
	l := NewAttemptLimiter(1, time.Hour, 10)

	if !l.AllowAttempt("register", "a@example.com", "1.2.3.4") {
		t.Fatal("first register attempt should be allowed")
	}
	if l.AllowAttempt("register", "a@example.com", "1.2.3.4") {
		t.Fatal("expected refusal: register's own limit for this email is already spent")
	}
	if !l.AllowAttempt("login", "a@example.com", "1.2.3.4") {
		t.Fatal("login should have its own budget for this email, unaffected by register")
	}
}

func TestAttemptLimiter_BoundedKeyCount(t *testing.T) {
	l := NewAttemptLimiter(1, time.Hour, 3)

	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("key-%d", i)
		if !l.allow(key) {
			t.Fatalf("expected key %d to be allowed under the cap", i)
		}
	}
	if len(l.windows) != 3 {
		t.Fatalf("window count = %d, want 3", len(l.windows))
	}

	if l.allow("key-3") {
		t.Fatal("expected a new key past the cap to be refused rather than allocated")
	}
	if len(l.windows) != 3 {
		t.Fatalf("window count after refusal = %d, want 3 (cap held)", len(l.windows))
	}
}

func TestAttemptLimiter_SweepEvictsExpiredWindows(t *testing.T) {
	l := NewAttemptLimiter(1, 10*time.Millisecond, 2)

	l.allow("a")
	l.allow("b")
	if len(l.windows) != 2 {
		t.Fatalf("window count = %d, want 2", len(l.windows))
	}

	time.Sleep(20 * time.Millisecond)

	// "c" is a distinct third key; it can only be admitted if the sweep
	// evicted the expired "a"/"b" windows first, since the map is at its cap.
	if !l.allow("c") {
		t.Fatal("expected the sweep to evict expired windows and admit a new key")
	}
	if len(l.windows) != 1 {
		t.Fatalf("window count after sweep = %d, want 1", len(l.windows))
	}
}
