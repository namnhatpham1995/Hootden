"use client";

import { useEffect, useState } from "react";
import { API_ORIGIN, ApiError, getAuthConfig, login, register } from "@/lib/api";
import { Bear } from "@/components/mascots/Bear";

const MIN_PASSWORD_LENGTH = 8;

// Single signed-out screen -- no marketing site in this version, see
// design.md's Non-Goals.
export function SignedOutLanding() {
  const [mode, setMode] = useState<"login" | "register">("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [revealed, setRevealed] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [googleEnabled, setGoogleEnabled] = useState(false);

  useEffect(() => {
    getAuthConfig()
      .then((cfg) => setGoogleEnabled(cfg.googleEnabled))
      .catch(() => setGoogleEnabled(false));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    if (mode === "register" && password.length < MIN_PASSWORD_LENGTH) {
      setError(`Password must be at least ${MIN_PASSWORD_LENGTH} characters.`);
      return;
    }

    setSubmitting(true);
    try {
      await (mode === "register" ? register(email, password) : login(email, password));
      // Hard reload, not router.push: useDen() needs to re-run getCurrentUser()
      // from scratch now that the session cookie is set.
      // eslint-disable-next-line @next/next/no-location-assign-relative-destination
      window.location.href = "/";
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Try again.");
      setSubmitting(false);
    }
  }

  return (
    <main
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: "var(--space-6)",
        textAlign: "center",
        padding: "var(--space-8)",
      }}
    >
      <Bear size={96} />
      <h1 style={{ fontFamily: "var(--font-display)", fontSize: "2rem" }}>Hootden</h1>
      <p style={{ color: "var(--foreground-muted)", maxWidth: "28ch" }}>
        A small den for your notes, checklists, and plans.
      </p>

      <form
        onSubmit={handleSubmit}
        style={{
          display: "flex",
          flexDirection: "column",
          gap: "var(--space-3)",
          width: "100%",
          maxWidth: "320px",
        }}
      >
        <div style={fieldStyle}>
          <label htmlFor="email" style={labelStyle}>
            Email
          </label>
          <input
            id="email"
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
            style={inputStyle}
          />
        </div>
        <div style={fieldStyle}>
          <label htmlFor="password" style={labelStyle}>
            Password
          </label>
          <div style={passwordWrapperStyle}>
            <input
              id="password"
              type={revealed ? "text" : "password"}
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              autoComplete={mode === "register" ? "new-password" : "current-password"}
              style={{ ...inputStyle, paddingRight: "6.5rem" }}
            />
            <button
              type="button"
              onClick={() => setRevealed((r) => !r)}
              aria-pressed={revealed}
              style={revealToggleStyle}
            >
              {revealed ? "Hide password" : "Show password"}
            </button>
          </div>
        </div>
        {error && <p style={{ color: "var(--danger)", fontSize: "0.9rem" }}>{error}</p>}
        <button
          type="submit"
          disabled={submitting}
          style={{
            fontFamily: "var(--font-ui)",
            fontWeight: 600,
            padding: "var(--space-3) var(--space-6)",
            borderRadius: "var(--radius-pill)",
            background: "var(--accent)",
            color: "var(--accent-foreground)",
            border: "none",
            cursor: submitting ? "default" : "pointer",
            opacity: submitting ? 0.7 : 1,
          }}
        >
          {mode === "register" ? "Create account" : "Sign in"}
        </button>
      </form>

      <button
        type="button"
        onClick={() => {
          setMode(mode === "register" ? "login" : "register");
          setError(null);
        }}
        style={{
          background: "none",
          border: "none",
          color: "var(--secondary)",
          fontFamily: "var(--font-ui)",
          fontSize: "0.9rem",
          cursor: "pointer",
        }}
      >
        {mode === "register" ? "Already have an account? Sign in" : "New here? Create an account"}
      </button>

      {googleEnabled && (
        <a
          href={`${API_ORIGIN}/auth/google/start`}
          style={{
            fontFamily: "var(--font-ui)",
            fontWeight: 600,
            padding: "var(--space-3) var(--space-6)",
            borderRadius: "var(--radius-pill)",
            border: "1px solid var(--border)",
            color: "var(--foreground)",
          }}
        >
          Sign in with Google
        </a>
      )}
    </main>
  );
}

const fieldStyle: React.CSSProperties = {
  display: "flex",
  flexDirection: "column",
  gap: "var(--space-1)",
  textAlign: "left",
};

const labelStyle: React.CSSProperties = {
  fontFamily: "var(--font-ui)",
  fontSize: "0.85rem",
  color: "var(--foreground-muted)",
};

const passwordWrapperStyle: React.CSSProperties = {
  position: "relative",
};

const revealToggleStyle: React.CSSProperties = {
  position: "absolute",
  right: "var(--space-2)",
  top: "50%",
  transform: "translateY(-50%)",
  background: "none",
  border: "none",
  color: "var(--secondary)",
  fontFamily: "var(--font-ui)",
  fontSize: "0.75rem",
  cursor: "pointer",
  padding: "var(--space-1) var(--space-2)",
};

const inputStyle: React.CSSProperties = {
  font: "inherit",
  fontFamily: "var(--font-ui)",
  padding: "var(--space-3) var(--space-4)",
  borderRadius: "var(--radius-md)",
  border: "1px solid var(--border)",
  background: "var(--surface-raised)",
  color: "var(--foreground)",
};
