import createClient from "openapi-fetch";
import type { paths } from "./generated/schema.js";

export type { paths, components, operations } from "./generated/schema.js";

/** Raw generated client; the hand-written layer (token, errors, cursors) wraps it in later tasks. */
export function createRawClient(baseUrl: string, fetch?: typeof globalThis.fetch) {
  return createClient<paths>(fetch ? { baseUrl, fetch } : { baseUrl });
}
