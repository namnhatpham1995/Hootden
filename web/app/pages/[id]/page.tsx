"use client";

import { use } from "react";
import { Den } from "@/components/Den";

export default function PageRoute({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);

  return (
    <Den selectedId={id}>
      {(pages) => {
        const current = pages.find((p) => p.id === id);
        return (
          <div style={{ padding: "var(--space-8)", maxWidth: "70ch" }}>
            <h1 style={{ fontFamily: "var(--font-display)", fontSize: "2rem" }}>
              {current?.title || "Untitled"}
            </h1>
            <p style={{ color: "var(--foreground-muted)" }}>
              The page editor lands in a later task group -- for now this just confirms the page opened.
            </p>
          </div>
        );
      }}
    </Den>
  );
}
