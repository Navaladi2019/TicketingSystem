// Setup for the SDK integration tests, run by the stack case sdk_ts before them (not by
// `node --test`: the name has no .test). Flags the fields the tests filter on (ADR-006) as tenant
// A's admin. On a laptop: node --experimental-strip-types test/integration/flag-fields.ts
import { ADMIN_A, flagFilterable } from "./stack.ts";

await flagFilterable(ADMIN_A, "incident", ["short_description"]);
console.log("flagged incident.short_description filterable");
