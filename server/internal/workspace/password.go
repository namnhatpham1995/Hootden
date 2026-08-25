package workspace

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/namnhatpham1995/Hootden/server/internal/auth"
)

// ErrEmailTaken means the email is already in use by another account,
// whether it was created via Google sign-in or password registration.
var ErrEmailTaken = errors.New("email already in use")

// ErrAccountNotFound means no account has this email.
var ErrAccountNotFound = errors.New("account not found")

const uniqueViolation = "23505"

// PasswordStore adapts CreateUserWithPassword and FindUserByEmail to
// auth.PasswordStore, so the register/login handlers can persist accounts
// without the auth package depending on the workspace package.
type PasswordStore struct {
	Pool *pgxpool.Pool
}

func (s PasswordStore) CreateUserWithPassword(ctx context.Context, email, passwordHash string) (string, error) {
	userID, _, err := CreateUserWithPassword(ctx, s.Pool, email, passwordHash)
	if errors.Is(err, ErrEmailTaken) {
		return "", auth.ErrEmailTaken
	}
	return userID, err
}

func (s PasswordStore) FindUserByEmail(ctx context.Context, email string) (string, string, error) {
	userID, passwordHash, err := FindUserByEmail(ctx, s.Pool, email)
	if errors.Is(err, ErrAccountNotFound) {
		return "", "", auth.ErrAccountNotFound
	}
	return userID, passwordHash, err
}

// CreateUserWithPassword creates a new account with a password and its
// personal Den in one transaction, mirroring EnsureUserAndDen's new-user
// branch. Unlike Google's upsert, this fails on a duplicate email rather
// than resolving to the existing account -- registration and sign-in are
// deliberately not the same action here, since there's a password to check.
func CreateUserWithPassword(ctx context.Context, pool *pgxpool.Pool, email, passwordHash string) (userID, denID string, err error) {
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
			email, passwordHash,
		).Scan(&userID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
				return ErrEmailTaken
			}
			return err
		}
		return tx.QueryRow(ctx,
			`INSERT INTO workspaces (owner_id, personal) VALUES ($1, true) RETURNING id`,
			userID,
		).Scan(&denID)
	})
	if err != nil {
		return "", "", err
	}
	return userID, denID, nil
}

// FindUserByEmail looks up an account by email for the login handler to
// verify a password against. passwordHash is empty for a Google-only
// account, which the caller treats the same as no account found.
func FindUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (userID, passwordHash string, err error) {
	err = pool.QueryRow(ctx,
		`SELECT id, COALESCE(password_hash, '') FROM users WHERE email = $1`,
		email,
	).Scan(&userID, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrAccountNotFound
	}
	if err != nil {
		return "", "", err
	}
	return userID, passwordHash, nil
}
