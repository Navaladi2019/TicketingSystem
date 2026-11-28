# Smoke runs, 2026-11-26: harness check only, not evidence

These runs used a busy host: 2 s cells, transport floor p50 3.6–8.2 ms, so the
host gate failed. They show only that the TICK-1009 (`edges`) and TICK-1039
(`arms`) phases run end to end. Do not score anything from them.

- `edges-and-arms-run1.md` covers `edges` in full and `arms` up to arm (b). It
  stopped on a bug: arm (b) left the field placeholder unbound (42P18). The
  bug is fixed.
- `arms-run2.md` is `arms` in full after the fix. Its result was `--- PASS`.
