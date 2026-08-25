"use client";

import { useState } from "react";
import type { PageNode } from "@/lib/api";

type Placement = "before" | "inside" | "after";
type TreeRow = PageNode & { depth: number };

const ROOT = "__root__";

function key(parentId: string | undefined): string {
  return parentId ?? "";
}

function childrenOf(nodes: PageNode[], parentId: string): PageNode[] {
  return nodes.filter((n) => key(n.parent_id) === parentId).sort((a, b) => a.position - b.position);
}

function flatten(nodes: PageNode[]): TreeRow[] {
  const rows: TreeRow[] = [];
  function visit(parentId: string, depth: number) {
    for (const n of childrenOf(nodes, parentId)) {
      rows.push({ ...n, depth });
      visit(n.id, depth + 1);
    }
  }
  visit("", 0);
  return rows;
}

function countDescendants(nodes: PageNode[], id: string): number {
  const children = nodes.filter((n) => n.parent_id === id);
  return children.length + children.reduce((sum, c) => sum + countDescendants(nodes, c.id), 0);
}

export function PageTree({
  nodes,
  selectedId,
  onSelect,
  onCreate,
  onRename,
  onMove,
  onDelete,
}: {
  nodes: PageNode[];
  selectedId?: string;
  onSelect: (id: string) => void;
  onCreate: (parentId: string) => void;
  onRename: (id: string, title: string) => void;
  onMove: (id: string, parentId: string, position: number) => void;
  onDelete: (id: string) => void;
}) {
  const rows = flatten(nodes);
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<{ id: string; placement: Placement } | null>(null);
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState("");

  function finishRename(id: string) {
    const trimmed = renameValue.trim();
    if (trimmed && trimmed !== nodes.find((n) => n.id === id)?.title) {
      onRename(id, trimmed);
    }
    setRenamingId(null);
  }

  function handleDrop() {
    if (draggedId && dropTarget) {
      // Positions are always computed excluding the dragged node itself --
      // the backend reindexes against the list *after* removal (see
      // Move()'s doc comment), so counting it in would leave a gap when
      // reordering within the same parent.
      const others = nodes.filter((n) => n.id !== draggedId);
      if (dropTarget.id === ROOT) {
        onMove(draggedId, "", childrenOf(others, "").length);
      } else {
        const target = nodes.find((n) => n.id === dropTarget.id);
        if (target) {
          if (dropTarget.placement === "inside") {
            onMove(draggedId, target.id, childrenOf(others, target.id).length);
          } else {
            const parentId = key(target.parent_id);
            const siblings = childrenOf(others, parentId);
            const idx = siblings.findIndex((n) => n.id === target.id);
            onMove(draggedId, parentId, dropTarget.placement === "after" ? idx + 1 : idx);
          }
        }
      }
    }
    setDraggedId(null);
    setDropTarget(null);
  }

  return (
    <div onMouseUp={handleDrop} onMouseLeave={() => !draggedId && setDropTarget(null)}>
      {rows.map((row) => (
        <div
          key={row.id}
          onMouseDown={(e) => {
            if (renamingId !== row.id) {
              e.preventDefault();
              setDraggedId(row.id);
            }
          }}
          onMouseEnter={(e) => {
            if (!draggedId || draggedId === row.id) return;
            const rect = e.currentTarget.getBoundingClientRect();
            const relY = (e.clientY - rect.top) / rect.height;
            const placement: Placement = relY < 0.25 ? "before" : relY > 0.75 ? "after" : "inside";
            setDropTarget({ id: row.id, placement });
          }}
          style={{
            marginLeft: row.depth * 16,
            display: "flex",
            alignItems: "center",
            gap: "var(--space-1)",
            padding: "var(--space-1) var(--space-2)",
            borderRadius: "var(--radius-sm)",
            cursor: "pointer",
            userSelect: "none",
            background: selectedId === row.id ? "var(--surface-raised)" : undefined,
            opacity: draggedId === row.id ? 0.4 : 1,
            outline:
              dropTarget?.id === row.id && dropTarget.placement === "inside"
                ? "2px solid var(--secondary)"
                : "none",
            borderTop:
              dropTarget?.id === row.id && dropTarget.placement === "before"
                ? "2px solid var(--accent)"
                : "2px solid transparent",
            borderBottom:
              dropTarget?.id === row.id && dropTarget.placement === "after"
                ? "2px solid var(--accent)"
                : "2px solid transparent",
          }}
        >
          {renamingId === row.id ? (
            <input
              autoFocus
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onBlur={() => finishRename(row.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter") finishRename(row.id);
                if (e.key === "Escape") setRenamingId(null);
              }}
              style={{ flex: 1, font: "inherit", background: "var(--surface)", color: "var(--foreground)" }}
            />
          ) : (
            <span
              style={{ flex: 1, fontFamily: "var(--font-ui)" }}
              onClick={() => onSelect(row.id)}
              onDoubleClick={() => {
                setRenamingId(row.id);
                setRenameValue(row.title);
              }}
            >
              {row.title || "Untitled"}
            </span>
          )}
          <button
            aria-label="Add subpage"
            onClick={() => onCreate(row.id)}
            style={{ border: "none", background: "transparent", color: "var(--foreground-muted)", cursor: "pointer" }}
          >
            +
          </button>
          <button
            aria-label="Delete page"
            onClick={() => {
              const count = countDescendants(nodes, row.id);
              const message =
                count > 0
                  ? `Delete "${row.title || "Untitled"}" and ${count} nested page${count === 1 ? "" : "s"}?`
                  : `Delete "${row.title || "Untitled"}"?`;
              if (window.confirm(message)) {
                onDelete(row.id);
              }
            }}
            style={{ border: "none", background: "transparent", color: "var(--foreground-muted)", cursor: "pointer" }}
          >
            ×
          </button>
        </div>
      ))}
      <div
        onMouseEnter={() => draggedId && setDropTarget({ id: ROOT, placement: "inside" })}
        style={{
          height: "var(--space-6)",
          borderRadius: "var(--radius-sm)",
          outline: dropTarget?.id === ROOT ? "2px solid var(--secondary)" : "none",
        }}
      />
      <button
        onClick={() => onCreate("")}
        style={{
          fontFamily: "var(--font-ui)",
          border: "none",
          background: "transparent",
          color: "var(--foreground-muted)",
          textAlign: "left",
          padding: "var(--space-1) var(--space-2)",
          cursor: "pointer",
        }}
      >
        + New page
      </button>
    </div>
  );
}
