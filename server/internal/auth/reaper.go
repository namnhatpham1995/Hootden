package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionReapInterval is intentionally long because expiry is enforced by
// ResolveSession; reaping only keeps the sessions table from retaining dead
// rows indefinitely.
const SessionReapInterval = time.Hour

// ReapExpiredSessions removes sessions that can no longer be resolved.
func ReapExpiredSessions(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return err
}

// StartSessionReaper runs the cleanup loop until ctx is cancelled. A failed
// cycle is logged and the loop continues so a transient database error does
// not terminate the server. The returned channel closes when the loop stops.
func StartSessionReaper(ctx context.Context, pool *pgxpool.Pool, interval time.Duration, logger *log.Logger) <-chan struct{} {
	if interval <= 0 {
		panic("session reaper interval must be positive")
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := ReapExpiredSessions(ctx, pool); err != nil && !errors.Is(err, context.Canceled) {
					logger.Printf("session reaper cycle failed: %v", err)
				}
			}
		}
	}()
	return done
}
