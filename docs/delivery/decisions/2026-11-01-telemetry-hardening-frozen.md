## 2026-11-01: telemetry redaction frozen at its current bar
- **Decided by:** vknavaladi (human), in chat with the Technical Lead, choosing option C of three (keep as-is, relax to a Collector-only deny-list, freeze).
- **Decision:** the in-process span allow-list (`internal/platform/telemetry/filter.go`), the metric View and exemplar wrapper, the log deny-list and the Collector deny-list stay as they are. No new telemetry hardening tickets are opened for edge cases beyond what the existing tests cover.
- **Still required:** a change that adds a span attribute, a `nava.*` key, an instrumentation library or an OTel upgrade must keep `TestServerSpanAttributeKeysAreGolden`, `TestCollectorDenyListMatchesFilter` and the httpobs adversarial tests green. A new key goes on the allow-list only after review against the TICK-94 item 3 span id policy. ADR-001's rule that no client-controlled value leaves the process is unchanged.
- **Not chosen:** relaxing to ".NET-style" defaults (option B) would need an ADR-001 amendment and an accepted-risk row.
- **Trigger to revisit:** a production trace or log backend is chosen (TODO-003), or a real leak is found in exported telemetry.
- **Dissent:** none.
