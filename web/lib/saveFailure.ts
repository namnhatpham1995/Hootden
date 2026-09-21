// Explicit extension: this module is loaded directly by Node's native test
// runner (see saveFailure.test.ts) as well as bundled by Next, and Node's
// ESM resolver requires the real extension for relative imports.
import { ApiError } from "./api.ts";

// A save failure either could succeed on a later attempt (network hiccup,
// server error) or never will against the same request (a rejected body,
// an expired session). See design.md's classification table -- ApiError
// already carries the status that answers it, and a plain network failure
// (no ApiError at all, just fetch's TypeError) is always worth retrying.
export function isRetryableSaveFailure(err: unknown): boolean {
  if (err instanceof ApiError) {
    return err.status >= 500;
  }
  return true;
}
