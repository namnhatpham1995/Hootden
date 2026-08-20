package auth

import "context"

// UserResolver turns a verified Google identity into a user ID, creating
// the account (and whatever else must exist alongside it) on first sign-in.
// Defined here rather than implemented here so the accounts package doesn't
// need to depend on the workspaces package just to provision a Den --
// workspace.UserResolver satisfies this without auth importing it.
type UserResolver interface {
	ResolveUser(ctx context.Context, googleSub, email string) (userID string, err error)
}
