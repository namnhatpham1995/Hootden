-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    google_sub TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    personal BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX workspaces_owner_id_idx ON workspaces (owner_id);

CREATE TABLE pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    parent_id UUID REFERENCES pages (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    doc JSONB NOT NULL DEFAULT '{}'::jsonb,
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX pages_workspace_id_idx ON pages (workspace_id);

-- A plain UNIQUE(workspace_id, parent_id, position) would not catch
-- duplicate positions among top-level pages, since Postgres treats every
-- NULL parent_id as distinct. Two partial indexes cover both cases.
CREATE UNIQUE INDEX pages_root_position_idx ON pages (workspace_id, position) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX pages_child_position_idx ON pages (workspace_id, parent_id, position) WHERE parent_id IS NOT NULL;

-- +goose Down
DROP TABLE pages;
DROP TABLE workspaces;
DROP TABLE sessions;
DROP TABLE users;
