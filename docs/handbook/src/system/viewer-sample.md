---
title: "Viewer sample"
section: system
status: delivered
---

The viewer gives every diagram zoom, pan, reset and full screen. Authors write nothing extra. Remove this page when real diagrams exist.

## Mermaid

```mermaid
flowchart LR
  client[Client] --> core[nava-core]
  core --> db[(PostgreSQL)]
  core --> outbox[Outbox]
  outbox --> relay[Relay] --> temporal[Temporal]
```

## Excalidraw

![Client calls nava-core](../diagrams/sample.excalidraw.svg)

## Search target

The word flamingo appears only on this page, so search can be checked.
