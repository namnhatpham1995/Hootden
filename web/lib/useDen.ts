"use client";

import { useCallback, useEffect, useState } from "react";
import { getCurrentUser, listPages, type Me, type PageNode } from "@/lib/api";

// "error" is a third state alongside Me | null | undefined -- undefined
// means still loading and null means genuinely signed out, so a failed
// fetch needs its own value rather than collapsing into either (see
// design.md's "me gains a third state rather than a second").
export type MeState = Me | null | undefined | "error";

// Shared by the root route and /pages/[id]: both need to know whether
// there's a signed-in user (to decide whether to render the Den at all)
// and the flat page list (to build the sidebar tree), so the fetch
// sequence lives here once instead of being redone slightly differently
// in each route.
export function useDen() {
  const [me, setMe] = useState<MeState>(undefined);
  const [pages, setPages] = useState<PageNode[]>([]);

  const refreshPages = useCallback(async () => {
    setPages(await listPages());
  }, []);

  const loadMe = useCallback(() => {
    let cancelled = false;
    getCurrentUser()
      .then((user) => {
        if (cancelled) return;
        setMe(user);
        if (user) {
          listPages().then((list) => {
            if (!cancelled) setPages(list);
          });
        }
      })
      .catch(() => {
        if (!cancelled) setMe("error");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => loadMe(), [loadMe]);

  return { me, pages, refreshPages, retry: loadMe };
}
