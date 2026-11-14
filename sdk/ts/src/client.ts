import createOpenApiClient from "openapi-fetch";
import type { components, paths } from "./generated/schema.js";
import { NavaProblem, problemFromResponse } from "./errors.ts";
import { SDK_VERSION } from "./version.ts";

export type NavaRecord = components["schemas"]["Record"];

export interface ClientOptions {
  baseUrl: string;
  /** Called before each request, and once more after a 401. The token is never kept beyond the call. */
  getToken: () => Promise<string>;
  fetch?: typeof globalThis.fetch;
}

export interface ListOptions {
  /** `filter[<property>]=value`; an array becomes `filter[<property>][in]=a,b`. */
  filter?: Record<string, string | number | boolean | Array<string | number | boolean>>;
  pageSize?: number;
}

export interface NavaClient {
  records: {
    list(type: string, opts?: ListOptions): AsyncIterable<NavaRecord>;
    get(type: string, id: string): Promise<{ record: NavaRecord; version: number }>;
    create(type: string, data: Record<string, unknown>, opts?: { idempotencyKey?: string }): Promise<NavaRecord>;
    update(type: string, id: string, patch: Record<string, unknown>, opts: { version: number }): Promise<NavaRecord>;
  };
}

function serializeQuery(q: { limit?: number; cursor?: string; filter?: ListOptions["filter"] }): string {
  const p = new URLSearchParams();
  if (q.limit !== undefined) p.append("limit", String(q.limit));
  if (q.cursor !== undefined) p.append("cursor", q.cursor);
  for (const [prop, val] of Object.entries(q.filter ?? {})) {
    if (Array.isArray(val)) p.append(`filter[${prop}][in]`, val.join(","));
    else p.append(`filter[${prop}]`, String(val));
  }
  return p.toString();
}

export function createClient(opts: ClientOptions): NavaClient {
  const baseFetch = opts.fetch ?? globalThis.fetch.bind(globalThis);

  // Wraps every request: token before it, one retry after a 401 with a fresh token, SDK header always.
  // The retry is a clone of the original, so it carries the same Idempotency-Key.
  const authedFetch = async (input: Request): Promise<Response> => {
    const retry = input.clone();
    const send = async (req: Request) => {
      req.headers.set("Authorization", `Bearer ${await opts.getToken()}`);
      req.headers.set("Nava-SDK", `ts/${SDK_VERSION}`);
      return baseFetch(req);
    };
    let res = await send(input);
    void retry; // TICK-829 mutation demo: the 401 retry removed. DO NOT MERGE.
    // Throwing here, before openapi-fetch reads the body, keeps the problem body readable.
    if (!res.ok) throw await problemFromResponse(res);
    return res;
  };

  const api = createOpenApiClient<paths>({ baseUrl: opts.baseUrl, fetch: authedFetch });

  function check<T>(r: { data?: T; response: Response }): { data: T } {
    if (r.data === undefined) throw new NavaProblem({ status: r.response.status, code: "unknown", title: "empty response", detail: "" });
    return { data: r.data };
  }

  return {
    records: {
      async *list(type, o = {}) {
        let cursor: string | undefined;
        do {
          const q = { limit: o.pageSize, cursor, filter: o.filter };
          const r = await api.GET("/api/v1/records/{entity}", {
            params: { path: { entity: type } },
            querySerializer: () => serializeQuery(q),
          });
          const { data } = check(r);
          yield* data.items;
          cursor = data.next_cursor ?? undefined;
        } while (cursor);
      },

      async get(type, id) {
        const { data } = check(await api.GET("/api/v1/records/{entity}/{id}", { params: { path: { entity: type, id } } }));
        return { record: data, version: data.version };
      },

      async create(type, data, o = {}) {
        const key = o.idempotencyKey ?? globalThis.crypto.randomUUID();
        const r = await api.POST("/api/v1/records/{entity}", {
          params: { path: { entity: type }, header: { "Idempotency-Key": key } },
          body: { data },
        });
        return check(r).data;
      },

      async update(type, id, patch, o) {
        const r = await api.PATCH("/api/v1/records/{entity}/{id}", {
          params: {
            path: { entity: type, id },
            header: { "If-Match": `W/"${o.version}"`, "Idempotency-Key": globalThis.crypto.randomUUID() },
          },
          body: { data: patch },
          headers: { "Content-Type": "application/merge-patch+json" },
        });
        return check(r).data;
      },
    },
  };
}
