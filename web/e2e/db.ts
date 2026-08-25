import { Pool } from "pg";
import { createHash, randomBytes, randomUUID } from "node:crypto";

const pool = new Pool({
  connectionString:
    process.env.DATABASE_URL ?? "postgres://postgres:postgres@localhost:5432/hootden?sslmode=disable",
});

export type Seeded = { userId: string; denId: string; token: string };

// Mirrors server/internal/auth/session.go's hashToken (sha256, base64url,
// no padding) -- the server only ever sees this hash, never the raw token.
function hashToken(token: string): string {
  return createHash("sha256").update(token).digest("base64url");
}

// Inserts a user, their personal Den, and a valid session directly into
// Postgres, mirroring what a real Google sign-in produces server-side.
// Google's consent screen can't be driven by an automated browser, so E2E
// auth starts here instead of a bypass endpoint -- see design.md.
export async function seedSession(): Promise<Seeded> {
  const suffix = randomUUID();
  const {
    rows: [user],
  } = await pool.query<{ id: string }>(`INSERT INTO users (google_sub, email) VALUES ($1, $2) RETURNING id`, [
    `e2e-${suffix}`,
    `e2e-${suffix}@example.test`,
  ]);
  const {
    rows: [den],
  } = await pool.query<{ id: string }>(
    `INSERT INTO workspaces (owner_id, personal) VALUES ($1, true) RETURNING id`,
    [user.id],
  );
  const token = randomBytes(32).toString("base64url");
  await pool.query(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, now() + interval '1 day')`, [
    hashToken(token),
    user.id,
  ]);
  return { userId: user.id, denId: den.id, token };
}

export async function closeDb(): Promise<void> {
  await pool.end();
}
