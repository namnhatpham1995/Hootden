"use client";

import { useEffect, useRef, useState } from "react";
import { useEditor, EditorContent, type JSONContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import { ApiError, getPage, saveDoc } from "@/lib/api";
import { isRetryableSaveFailure } from "@/lib/saveFailure";

type SaveStatus = "loading" | "saved" | "saving" | "failed" | "rejected" | "signedOut";

const SAVE_DEBOUNCE_MS = 800;
const RETRY_BASE_MS = 1000;
const RETRY_MAX_MS = 16000;

function statusLabel(status: SaveStatus, rejectionReason: string | null): string {
  switch (status) {
    case "loading":
      return "";
    case "saving":
      return "Saving…";
    case "failed":
      return "Failed to save — retrying…";
    case "rejected":
      return `Could not be saved: ${rejectionReason ?? "unknown reason"}`;
    case "signedOut":
      return "Signed out — this change was not saved";
    case "saved":
      return "Saved";
  }
}

// Mount with a `key={pageId}` from the caller so navigating to a different
// page fully remounts this component -- simpler and safer than resetting
// the save/retry refs by hand on every pageId change.
export function Editor({ pageId }: { pageId: string }) {
  const [status, setStatus] = useState<SaveStatus>("loading");
  const [rejectionReason, setRejectionReason] = useState<string | null>(null);
  const dirtyRef = useRef(false);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryDelay = useRef(RETRY_BASE_MS);
  const latestDoc = useRef<JSONContent | null>(null);

  const editor = useEditor({
    extensions: [StarterKit, TaskList, TaskItem.configure({ nested: true })],
    immediatelyRender: false,
    editorProps: {
      attributes: { class: "editor-doc" },
    },
    onUpdate: ({ editor }) => {
      latestDoc.current = editor.getJSON();
      dirtyRef.current = true;
      setStatus("saving");
      scheduleSave(SAVE_DEBOUNCE_MS);
    },
  });

  function scheduleSave(delay: number) {
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(runSave, delay);
  }

  async function runSave() {
    if (!dirtyRef.current) return;
    const doc = latestDoc.current;
    try {
      await saveDoc(pageId, doc);
      retryDelay.current = RETRY_BASE_MS;
      // Only clear dirty/saved if nothing changed while this save was in flight.
      if (latestDoc.current === doc) {
        dirtyRef.current = false;
        setStatus("saved");
      }
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        // The session is gone. Stop retrying against it, but don't
        // navigate -- that's apiFetch's default for reads, not writes (see
        // design.md) -- a redirect here would silently drop what's on screen.
        retryDelay.current = RETRY_BASE_MS;
        setStatus("signedOut");
      } else if (isRetryableSaveFailure(err)) {
        setStatus("failed");
        scheduleSave(retryDelay.current);
        retryDelay.current = Math.min(retryDelay.current * 2, RETRY_MAX_MS);
      } else {
        // This exact request will never succeed -- retrying it forever
        // would just show "retrying…" over a document that can't be saved.
        retryDelay.current = RETRY_BASE_MS;
        setRejectionReason(err instanceof Error ? err.message : "unknown reason");
        setStatus("rejected");
      }
    }
  }

  useEffect(() => {
    if (!editor) return;
    let cancelled = false;
    getPage(pageId).then((page) => {
      if (cancelled) return;
      editor.commands.setContent((page.doc as JSONContent) ?? { type: "doc", content: [] });
      dirtyRef.current = false;
      setStatus("saved");
    });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- remounted per pageId via caller's key
  }, [editor]);

  useEffect(() => {
    function handler(e: BeforeUnloadEvent) {
      if (dirtyRef.current) {
        e.preventDefault();
      }
    }
    window.addEventListener("beforeunload", handler);
    return () => {
      window.removeEventListener("beforeunload", handler);
      if (saveTimer.current) clearTimeout(saveTimer.current);
      // The request outlives the component -- an in-flight fetch isn't
      // cancelled by unmounting -- so fire the pending/retrying save instead
      // of just dropping its timer. One attempt only: there's no component
      // left to retry from, so a failure here has nowhere left to go but an
      // alert (see design.md's "flush on unmount" decision).
      if (dirtyRef.current) {
        const doc = latestDoc.current;
        saveDoc(pageId, doc).catch(() => {
          window.alert("A change to this page could not be saved.");
        });
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- remounted per pageId via caller's key
  }, []);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-2)" }}>
      <div style={{ display: "flex", gap: "var(--space-1)", flexWrap: "wrap" }}>
        <ToolbarButton label="H1" onClick={() => editor?.chain().focus().toggleHeading({ level: 1 }).run()} />
        <ToolbarButton label="H2" onClick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()} />
        <ToolbarButton label="H3" onClick={() => editor?.chain().focus().toggleHeading({ level: 3 }).run()} />
        <ToolbarButton label="Bold" onClick={() => editor?.chain().focus().toggleBold().run()} />
        <ToolbarButton label="Italic" onClick={() => editor?.chain().focus().toggleItalic().run()} />
        <ToolbarButton label="Strike" onClick={() => editor?.chain().focus().toggleStrike().run()} />
        <ToolbarButton label="Code" onClick={() => editor?.chain().focus().toggleCode().run()} />
        <ToolbarButton label="Bullets" onClick={() => editor?.chain().focus().toggleBulletList().run()} />
        <ToolbarButton label="Numbered" onClick={() => editor?.chain().focus().toggleOrderedList().run()} />
        <ToolbarButton label="Checklist" onClick={() => editor?.chain().focus().toggleTaskList().run()} />
        <ToolbarButton label="Quote" onClick={() => editor?.chain().focus().toggleBlockquote().run()} />
        <ToolbarButton label="Code block" onClick={() => editor?.chain().focus().toggleCodeBlock().run()} />
        <ToolbarButton label="Divider" onClick={() => editor?.chain().focus().setHorizontalRule().run()} />
        <span
          style={{
            marginLeft: "auto",
            alignSelf: "center",
            fontFamily: "var(--font-ui)",
            fontSize: "0.85rem",
            color:
              status === "failed"
                ? "var(--secondary)"
                : status === "rejected" || status === "signedOut"
                  ? "var(--danger)"
                  : "var(--foreground-muted)",
          }}
        >
          {statusLabel(status, rejectionReason)}
        </span>
      </div>
      <EditorContent editor={editor} />
    </div>
  );
}

function ToolbarButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        fontFamily: "var(--font-ui)",
        fontSize: "0.85rem",
        border: "1px solid var(--border)",
        borderRadius: "var(--radius-sm)",
        padding: "var(--space-1) var(--space-2)",
        background: "var(--surface-raised)",
        color: "var(--foreground)",
        cursor: "pointer",
      }}
    >
      {label}
    </button>
  );
}
