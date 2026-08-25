"use client";

import type { ReactNode } from "react";
import { useRouter } from "next/navigation";
import { createPage, deletePage, movePage, renamePage, type PageNode } from "@/lib/api";
import { useDen } from "@/lib/useDen";
import { SignedOutLanding } from "@/components/SignedOutLanding";
import { Shell } from "@/components/Shell";
import { PageTree } from "@/components/PageTree";
import { Bear } from "@/components/mascots/Bear";

// Wraps Shell + PageTree with the fetch/create/rename/move/delete wiring
// shared by the root route and /pages/[id] -- the content region is the
// only thing that differs between them, so it's a render-prop over the
// current page list rather than each route re-fetching separately.
export function Den({
  selectedId,
  children,
}: {
  selectedId?: string;
  children: (pages: PageNode[]) => ReactNode;
}) {
  const { me, pages, refreshPages } = useDen();
  const router = useRouter();

  if (me === undefined) {
    return null;
  }
  if (me === null) {
    return <SignedOutLanding />;
  }

  async function handleCreate(parentId: string) {
    const node = await createPage(parentId, "");
    await refreshPages();
    router.push(`/pages/${node.id}`);
  }

  async function handleRename(id: string, title: string) {
    await renamePage(id, title);
    await refreshPages();
  }

  async function handleMove(id: string, parentId: string, position: number) {
    await movePage(id, parentId, position);
    await refreshPages();
  }

  async function handleDelete(id: string) {
    await deletePage(id);
    await refreshPages();
    if (id === selectedId) {
      router.push("/");
    }
  }

  return (
    <Shell
      tree={
        <PageTree
          nodes={pages}
          selectedId={selectedId}
          onSelect={(id) => router.push(`/pages/${id}`)}
          onCreate={handleCreate}
          onRename={handleRename}
          onMove={handleMove}
          onDelete={handleDelete}
        />
      }
    >
      {pages.length === 0 ? (
        <div
          style={{
            flex: 1,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            justifyContent: "center",
            gap: "var(--space-4)",
            padding: "var(--space-8)",
            textAlign: "center",
          }}
        >
          <Bear size={96} sleeping />
          <p style={{ color: "var(--foreground-muted)" }}>Your den is empty.</p>
          <button
            onClick={() => handleCreate("")}
            style={{
              fontFamily: "var(--font-ui)",
              fontWeight: 600,
              padding: "var(--space-3) var(--space-6)",
              borderRadius: "var(--radius-pill)",
              background: "var(--accent)",
              color: "var(--accent-foreground)",
              border: "none",
              cursor: "pointer",
            }}
          >
            Create your first page
          </button>
        </div>
      ) : (
        children(pages)
      )}
    </Shell>
  );
}
