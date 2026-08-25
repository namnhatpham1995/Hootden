"use client";

import { use } from "react";
import { Den } from "@/components/Den";
import { Editor } from "@/components/Editor";

export default function PageRoute({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);

  return (
    <Den selectedId={id}>
      {(pages) => {
        const current = pages.find((p) => p.id === id);
        return (
          <div style={{ padding: "var(--space-8)", maxWidth: "70ch", width: "100%" }}>
            <h1 style={{ fontFamily: "var(--font-display)", fontSize: "2rem", marginBottom: "var(--space-4)" }}>
              {current?.title || "Untitled"}
            </h1>
            <Editor key={id} pageId={id} />
          </div>
        );
      }}
    </Den>
  );
}
