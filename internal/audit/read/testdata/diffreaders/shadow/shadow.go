// Package shadow is a decoy for the per-file import scope check of the
// TICK-541 diff reader test: shadowed.go imports it under the local name
// audited, so its LogTable must not leak into readers.go, which imports the
// real internal/db/audited under the same name (TICK-551 item 3).
package shadow

// LogTable has the same name as audited.LogTable but is another table.
const LogTable = "event_log"
