"use client";

import { api, type Me } from "@/lib/api";
import { Bear } from "@/components/mascots/Bear";

type Theme = "light" | "dark";

// The DOM attribute (set synchronously by the inline script in layout.tsx
// before first paint) is the source of truth, not React state -- that's
// what avoids both a hydration mismatch and an unstyled flash on reload.
function currentTheme(): Theme {
  const explicit = document.documentElement.dataset.theme;
  if (explicit === "light" || explicit === "dark") {
    return explicit;
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function ThemeToggle() {
  function toggle() {
    const next: Theme = currentTheme() === "dark" ? "light" : "dark";
    document.documentElement.dataset.theme = next;
    window.localStorage.setItem("theme", next);
  }

  return (
    <button
      onClick={toggle}
      style={{
        fontFamily: "var(--font-ui)",
        border: "1px solid var(--border)",
        borderRadius: "var(--radius-pill)",
        padding: "var(--space-1) var(--space-3)",
        background: "var(--surface-raised)",
        color: "var(--foreground)",
      }}
    >
      Toggle theme
    </button>
  );
}

// App shell: sidebar plus content region. No workspace switcher, create, or
// delete affordance -- there is exactly one Den per account in this
// version. The page tree (task group 9) fills the sidebar later.
export function Shell({ me }: { me: Me }) {
  async function signOut() {
    await api.post("/auth/signout");
    // Hard reload so nothing signed-in stays in memory or component state.
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.href = "/";
  }

  return (
    <div style={{ display: "flex", minHeight: "100vh" }}>
      <aside
        style={{
          width: 240,
          borderRight: "1px solid var(--border)",
          padding: "var(--space-4)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--space-4)",
        }}
      >
        <span style={{ fontFamily: "var(--font-display)", fontSize: "1.25rem" }}>Hootden</span>
        <div style={{ flex: 1 }} />
        <ThemeToggle />
        <button
          onClick={signOut}
          style={{
            fontFamily: "var(--font-ui)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius-md)",
            padding: "var(--space-2) var(--space-3)",
            background: "transparent",
            color: "var(--foreground)",
          }}
        >
          Sign out
        </button>
      </aside>
      <main style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: "var(--space-4)" }}>
        <Bear size={96} />
        <p style={{ color: "var(--foreground-muted)" }}>Signed in as {me.email}</p>
      </main>
    </div>
  );
}
