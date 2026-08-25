package auth

import (
	"context"
	"errors"
)

// ErrEmailTaken means the email is already in use by another account.
var ErrEmailTaken = errors.New("email already in use")

// ErrAccountNotFound means no account has this email.
var ErrAccountNotFound = errors.New("account not found")

// PasswordStore persists password-authenticated accounts. Defined here
// rather than implemented here so the accounts package doesn't need to
// depend on the workspaces package just to provision a Den --
// workspace.PasswordStore satisfies this without auth importing it.
type PasswordStore interface {
	CreateUserWithPassword(ctx context.Context, email, passwordHash string) (userID string, err error)
	FindUserByEmail(ctx context.Context, email string) (userID, passwordHash string, err error)
}
