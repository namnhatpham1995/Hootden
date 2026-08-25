import { API_ORIGIN } from "@/lib/api";
import { Bear } from "@/components/mascots/Bear";

// Single signed-out screen -- no marketing site in this version, see
// design.md's Non-Goals.
export function SignedOutLanding() {
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
      <a
        href={`${API_ORIGIN}/auth/google/start`}
        style={{
          fontFamily: "var(--font-ui)",
          fontWeight: 600,
          padding: "var(--space-3) var(--space-6)",
          borderRadius: "var(--radius-pill)",
          background: "var(--accent)",
          color: "var(--accent-foreground)",
        }}
      >
        Sign in with Google
      </a>
    </main>
  );
}
