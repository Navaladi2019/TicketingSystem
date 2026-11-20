// Package configsurface is the upgrade-compatibility suite (ADR-008 X2,
// M-X2; PLAN-009 P9-02). It holds no production code: its tests fail, naming
// the item, when a change removes a workflow step type or a type-definition
// keyword that tenant configuration can use, or stops accepting a stored
// fixture that uses it.
//
// testdata/inventory.json lists the configuration surface tenants can use.
// The tests compare it with the sources of truth (internal/workflow/def's
// schema.json and Parse, internal/metadata's Compile) in four checks:
//
//  1. removal: every inventory item is still in the source of truth;
//  2. unlisted addition: every item in the source of truth is in the
//     inventory (additions are allowed, they must be listed, ADR-008 D12);
//  3. still accepted: every fixture under testdata/fixtures still parses or
//     compiles, and every inventory item is used by at least one fixture;
//  4. deprecated items are still listed and still accepted.
//
// A reserved item (inventory "reserved") is one the source of truth names but
// Compile refuses, such as x-nava-searchable (ADR-006 D8). Check 2 treats it
// as listed; checks 1, 3 and 4 skip it; TestReservedStillRefused fails once
// Compile accepts it, so it then moves into the inventory proper.
//
// Check 5, the baseline, is cmd/compat: it fails when an item listed on the
// base branch's inventory is missing from HEAD's, so an item removed from both
// the code and the inventory is still caught. It runs in make lint and
// task lint.
//
// type_definition.keywords is hand-curated (internal/metadata accepts any JSON
// Schema 2020-12 keyword and has no allow-list to read): every keyword the
// seed templates use plus the marked-property keywords. Check 2 covers only
// what can be enumerated, the x-nava-* extensions and the fields of the
// type-definition document; removing a standard keyword is caught by check 3.
//
// No production package imports this one.
package configsurface
