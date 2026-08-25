"use client";

import type { ReactNode } from "react";
import { api } from "@/lib/api";

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

// App shell: sidebar plus content region. No workspace switcher -- there is
// exactly one Den per account in this version. `tree` fills the sidebar
// below the wordmark, `children` fills the content region; callers (the
// root route and /pages/[id]) own what actually goes in each slot.
export function Shell({ tree, children }: { tree?: ReactNode; children?: ReactNode }) {
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
        <div style={{ flex: 1, overflowY: "auto" }}>{tree}</div>
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
      <main style={{ flex: 1, display: "flex", flexDirection: "column" }}>{children}</main>
    </div>
  );
}
