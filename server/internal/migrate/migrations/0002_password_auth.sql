-- +goose Up
ALTER TABLE users ALTER COLUMN google_sub DROP NOT NULL;
ALTER TABLE users ADD COLUMN password_hash TEXT;
ALTER TABLE users ADD CONSTRAINT users_has_auth_method
    CHECK (google_sub IS NOT NULL OR password_hash IS NOT NULL);
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

-- +goose Down
ALTER TABLE users DROP CONSTRAINT users_email_key;
ALTER TABLE users DROP CONSTRAINT users_has_auth_method;
ALTER TABLE users DROP COLUMN password_hash;
ALTER TABLE users ALTER COLUMN google_sub SET NOT NULL;
