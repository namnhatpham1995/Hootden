package auth

import (
	"bytes"
	"context"
	"io"
	"log"
	"strings"
	"testing"
	"time"
)

func TestSessionReaperDeletesExpiredRowsAndKeepsLiveRows(t *testing.T) {
	pool := testPool(t)
	userID := createTestUser(t)

	expiredToken, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken expired: %v", err)
	}
	liveToken, err := randomToken()
	if err != nil {
		t.Fatalf("randomToken live: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3), ($4, $2, $5)
	`, hashToken(expiredToken), userID, time.Now().Add(-time.Hour), hashToken(liveToken), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("insert test sessions: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := StartSessionReaper(ctx, pool, 10*time.Millisecond, log.New(io.Discard, "", 0))
	t.Cleanup(func() {
		cancel()
		<-done
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		var expiredCount, liveCount int
		err := pool.QueryRow(t.Context(), `SELECT count(*) FILTER (WHERE token_hash = $1), count(*) FILTER (WHERE token_hash = $2) FROM sessions`, hashToken(expiredToken), hashToken(liveToken)).Scan(&expiredCount, &liveCount)
		if err != nil {
			t.Fatalf("count test sessions: %v", err)
		}
		if expiredCount == 0 {
			if liveCount != 1 {
				t.Fatalf("live session count = %d, want 1", liveCount)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expired session remained after reaper cycle")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSessionReaperStopsWhenContextIsCancelled(t *testing.T) {
	pool := testPool(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := StartSessionReaper(ctx, pool, time.Hour, log.New(io.Discard, "", 0))

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("session reaper did not stop after context cancellation")
	}
}

func TestSessionReaperLogsFailureAndKeepsRunning(t *testing.T) {
	pool := testPool(t)
	pool.Close()

	var logs bytes.Buffer
	ctx, cancel := context.WithCancel(t.Context())
	done := StartSessionReaper(ctx, pool, 10*time.Millisecond, log.New(&logs, "", 0))
	defer func() {
		cancel()
		<-done
	}()

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(logs.String(), "session reaper cycle failed") {
		select {
		case <-done:
			t.Fatal("session reaper stopped after a failed cycle")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("session reaper did not log a failed cycle")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
