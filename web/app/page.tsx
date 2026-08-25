"use client";

import { Den } from "@/components/Den";

export default function Home() {
  return (
    <Den>
      {() => (
        <div
          style={{
            flex: 1,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: "var(--foreground-muted)",
          }}
        >
          Select a page, or create a new one.
        </div>
      )}
    </Den>
  );
}
