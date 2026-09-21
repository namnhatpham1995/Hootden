import { test } from "node:test";
import assert from "node:assert/strict";
// Explicit .ts extensions: this file runs under Node's native type-stripping
// test runner (`node --test`), not the Next.js bundler, and Node's ESM
// resolver -- unlike tsc's "bundler" mode -- requires the real extension.
import { ApiError } from "./api.ts";
import { isRetryableSaveFailure } from "./saveFailure.ts";

test("a 503 schedules a retry", () => {
  assert.equal(isRetryableSaveFailure(new ApiError(503, "unavailable")), true);
});

test("a 400 does not schedule a retry", () => {
  assert.equal(isRetryableSaveFailure(new ApiError(400, "bad request")), false);
});

test("a 404 does not schedule a retry", () => {
  assert.equal(isRetryableSaveFailure(new ApiError(404, "not found")), false);
});

test("a network failure (no ApiError at all) schedules a retry", () => {
  assert.equal(isRetryableSaveFailure(new TypeError("fetch failed")), true);
});
