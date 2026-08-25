"use client";

import { useCallback, useEffect, useState } from "react";
import { getCurrentUser, listPages, type Me, type PageNode } from "@/lib/api";

// Shared by the root route and /pages/[id]: both need to know whether
// there's a signed-in user (to decide whether to render the Den at all)
// and the flat page list (to build the sidebar tree), so the fetch
// sequence lives here once instead of being redone slightly differently
// in each route.
export function useDen() {
  const [me, setMe] = useState<Me | null | undefined>(undefined);
  const [pages, setPages] = useState<PageNode[]>([]);

  const refreshPages = useCallback(async () => {
    setPages(await listPages());
  }, []);

  useEffect(() => {
    let cancelled = false;
    getCurrentUser().then((user) => {
      if (cancelled) return;
      setMe(user);
      if (user) {
        listPages().then((list) => {
          if (!cancelled) setPages(list);
        });
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return { me, pages, refreshPages };
}
