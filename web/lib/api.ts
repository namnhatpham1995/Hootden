export const API_ORIGIN = process.env.NEXT_PUBLIC_API_ORIGIN ?? "http://localhost:8080";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

// apiFetch calls the Go API directly (never through a Next.js route
// handler -- see design.md) with the session cookie attached. A 401 always
// throws; redirectOn401 additionally sends the caller straight to the
// signed-out landing screen, for the reads where a dead session should
// bounce the whole app rather than be handled locally. Off by default --
// see design.md's "redirect becomes opt-in" -- because most callers (every
// write) need to keep whatever the person was doing on screen and handle
// the 401 themselves instead of losing it to a hard navigation.
async function apiFetch<T>(path: string, init?: RequestInit, opts: { redirectOn401?: boolean } = {}): Promise<T> {
  const res = await fetch(`${API_ORIGIN}${path}`, {
    ...init,
    credentials: "include",
    headers: { "Content-Type": "application/json", ...init?.headers },
  });

  if (res.status === 401) {
    if (opts.redirectOn401 && typeof window !== "undefined") {
      // A hard reload, not router.push: apiFetch is a plain function called
      // from anywhere, not a component with a router instance, and losing
      // all client state on sign-out is the correct behaviour anyway.
      // eslint-disable-next-line @next/next/no-location-assign-relative-destination
      window.location.href = "/";
    }
    throw new ApiError(401, "not signed in");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}) as { error?: string });
    throw new ApiError(res.status, body.error ?? `request failed (${res.status})`);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return res.json();
}

export const api = {
  // GET is the only verb that opts into the redirect -- a dead session on a
  // read means there's nothing on screen worth preserving over bouncing to
  // the landing screen; every write handles its own 401 (see saveDoc).
  get: <T>(path: string) => apiFetch<T>(path, undefined, { redirectOn401: true }),
  post: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, { method: "POST", body: body !== undefined ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    apiFetch<T>(path, { method: "PATCH", body: body !== undefined ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string) => apiFetch<T>(path, { method: "DELETE" }),
};

export type Me = {
  id: string;
  email: string;
  workspace: { id: string };
};

// A 401 from login/register is an expected "wrong credentials" outcome, not
// a dead session -- api.post doesn't request the redirect, so this needs no
// special handling beyond the ApiError every other failure already throws.
export function register(email: string, password: string): Promise<void> {
  return api.post<void>("/auth/register", { email, password });
}

export function login(email: string, password: string): Promise<void> {
  return api.post<void>("/auth/login", { email, password });
}

export async function getAuthConfig(): Promise<{ googleEnabled: boolean }> {
  const res = await fetch(`${API_ORIGIN}/auth/config`, { credentials: "include" });
  if (!res.ok) throw new ApiError(res.status, "failed to load sign-in options");
  return res.json();
}

// getCurrentUser is the one caller that must NOT request apiFetch's 401
// redirect -- an unauthenticated response here is the expected signed-out
// state, not an error to bounce away from, since this is what decides
// whether to show the landing screen or the Den in the first place.
export async function getCurrentUser(): Promise<Me | null> {
  try {
    return await apiFetch<Me>("/me");
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) return null;
    throw err;
  }
}

// PageNode is the flat list shape /pages returns -- no nested children, the
// tree is assembled client-side from parent_id + position (see PageTree).
// parent_id is omitted by the backend for root pages (Go's omitempty),
// never sent as null, so it's optional here rather than nullable.
export type PageNode = {
  id: string;
  parent_id?: string;
  title: string;
  position: number;
};

export type PageDoc = PageNode & { doc: unknown; updated_at: string };

export function listPages(): Promise<PageNode[]> {
  return api.get<PageNode[]>("/pages");
}

export function getPage(id: string): Promise<PageDoc> {
  return api.get<PageDoc>(`/pages/${id}`);
}

export function createPage(parentId: string, title: string): Promise<PageNode> {
  return api.post<PageNode>("/pages", { parent_id: parentId, title });
}

// The PATCH endpoint returns 204 with no body for every update, so these
// resolve to void -- callers refetch the list/page to see the result.
export function renamePage(id: string, title: string): Promise<void> {
  return api.patch<void>(`/pages/${id}`, { title });
}

// movePage always sends parent_id together with position -- the backend
// only treats this as a move (vs. a plain rename) when position is present.
export function movePage(id: string, parentId: string, position: number): Promise<void> {
  return api.patch<void>(`/pages/${id}`, { parent_id: parentId, position });
}

export function saveDoc(id: string, doc: unknown): Promise<void> {
  return api.patch<void>(`/pages/${id}`, { doc });
}

export function deletePage(id: string): Promise<void> {
  return api.delete<void>(`/pages/${id}`);
}
