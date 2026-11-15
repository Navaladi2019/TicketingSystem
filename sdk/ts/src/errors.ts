/** Field-level detail of an RFC 9457 problem (`errors[]`). */
export interface FieldError {
  pointer: string;
  rule?: string;
}

export interface ProblemInit {
  status: number;
  code: string;
  title: string;
  detail: string;
  fields?: FieldError[];
}

/** An RFC 9457 problem answered by the API. `code` is registered in api/errors.yaml. */
export class NavaProblem extends Error {
  readonly status: number;
  readonly code: string;
  readonly title: string;
  readonly detail: string;
  readonly fields?: FieldError[];

  constructor(init: ProblemInit) {
    super(init.detail || init.title || init.code);
    this.name = new.target.name;
    this.status = init.status;
    this.code = init.code;
    this.title = init.title;
    this.detail = init.detail;
    if (init.fields) this.fields = init.fields;
  }
}

/** 412: the `If-Match` version is not the newest. */
export class VersionConflict extends NavaProblem {}

/** 429: `retryAfter` is the advised wait in seconds (undefined when the header is absent). */
export class RateLimited extends NavaProblem {
  readonly retryAfter: number | undefined;

  constructor(init: ProblemInit, retryAfter?: number) {
    super(init);
    this.retryAfter = retryAfter;
  }
}

function parseRetryAfter(value: string | null): number | undefined {
  if (value === null) return undefined;
  const seconds = Number(value);
  if (Number.isFinite(seconds) && seconds >= 0) return seconds;
  const at = Date.parse(value);
  return Number.isNaN(at) ? undefined : Math.max(0, Math.ceil((at - Date.now()) / 1000));
}

/** Builds the typed error for a non-2xx response. Never throws; a non-JSON body still yields a NavaProblem. */
export async function problemFromResponse(res: Response): Promise<NavaProblem> {
  let body: Record<string, unknown> = {};
  try {
    const parsed: unknown = JSON.parse(await res.text());
    if (parsed && typeof parsed === "object") body = parsed as Record<string, unknown>;
  } catch {
    // keep the empty body
  }
  const str = (v: unknown, d: string) => (typeof v === "string" ? v : d);
  const init: ProblemInit = {
    status: res.status,
    code: str(body.code, "unknown"),
    title: str(body.title, res.statusText),
    detail: str(body.detail, ""),
  };
  if (Array.isArray(body.errors)) init.fields = body.errors as FieldError[];
  if (res.status === 412) return new VersionConflict(init);
  if (res.status === 429) return new RateLimited(init, parseRetryAfter(res.headers.get("Retry-After")));
  return new NavaProblem(init);
}
