import type { components } from "./generated/schema.js";

type RecordList = components["schemas"]["RecordList"];

export type TotalState = RecordList["total_state"];
export interface TotalView {
  text: string;
  state: TotalState;
  value: number | null;
}

const CAP = 1000;
const fmt = new Intl.NumberFormat("en-US");

/**
 * Text for a list's total (ADR-006-A2 A2-1). Never renders `unknown` as "0".
 * page: the RecordList (total and total_state only).
 * listed: matches already shown for this result set, including this page.
 * best: the view returned for an earlier page of the same result set.
 */
export function describeTotal(
  page: Pick<RecordList, "total" | "total_state">,
  listed?: number,
  best?: TotalView,
): TotalView {
  if (page.total_state === "unknown") {
    if (best && best.state !== "unknown") return best;
    if (typeof listed === "number" && listed >= 1) {
      return { text: `${fmt.format(listed)}+`, state: "unknown", value: null };
    }
    if (best) return best;
    return { text: "many", state: "unknown", value: null };
  }
  if (page.total === null) {
    throw new TypeError(`total_state ${page.total_state} requires a non-null total`);
  }
  if (page.total_state === "capped") {
    return { text: `${fmt.format(CAP)}+`, state: "capped", value: CAP };
  }
  return { text: fmt.format(page.total), state: "exact", value: page.total };
}
