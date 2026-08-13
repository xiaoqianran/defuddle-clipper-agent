# Architecture

## Product boundary

The browser extension is a sensor. The local desktop application is the product.

```text
┌──────────────────────── Browser ────────────────────────┐
│                                                        │
│ rendered/authenticated DOM                             │
│        ↓                                               │
│ navigation observer                                    │
│        ↓                                               │
│ debounce + DOM stability                               │
│        ↓                                               │
│ Defuddle                                               │
│        ↓                                               │
│ ContentPacket v1                                       │
│        ↓                                               │
│ background delivery + retry queue                      │
└────────────────────────┬───────────────────────────────┘
                         │ localhost HTTP
                         ▼
┌──────────────────── Local Go Core ─────────────────────┐
│                                                       │
│ capture endpoint                                      │
│      ↓                                                │
│ CaptureService                                        │
│      ├─ raw-first Store                               │
│      ├─ current-page / Follow Browser state           │
│      ├─ archive/index services                        │
│      └─ optional Analyzer                             │
└────────────────────────┬──────────────────────────────┘
                         │
                         ▼
┌──────────────────── Desktop Application ───────────────┐
│                                                       │
│ History        Reader                 AI / Notes       │
│                                                       │
│ Follow Browser | Archive All | Search | Settings      │
└───────────────────────────────────────────────────────┘
```

## Browser capture model

Normal operation must not require a popup click.

The extension observes:

- normal page loads;
- active-tab changes;
- SPA navigation (`pushState`, `replaceState`, `popstate`);
- URL changes that do not cause full reloads.

A capture coordinator prevents noise:

```text
navigation signal
    ↓
debounce
    ↓
wait for usable/stable DOM
    ↓
Defuddle.parseAsync()
    ↓
canonical URL + content fingerprint
    ↓
new meaningful page?
  ├─ no  → ignore
  └─ yes → submit ContentPacket
```

Scrolling, ads, lazy widgets and unrelated DOM mutations must not generate a capture flood.

Manual capture remains a fallback/debug action, not the primary interaction.

## Why the browser stays thin

The extension is good at:

- seeing the rendered page, including authenticated content;
- detecting navigation and active-tab state;
- running Defuddle against the live DOM;
- forwarding selections/transcripts/metadata;
- buffering failed deliveries.

The extension should not own:

- the primary reading UI;
- model credentials;
- local filesystem orchestration;
- indexing/search databases;
- large durable queues;
- long-running processing;
- knowledge relationships.

## Desktop application

Preferred target stack:

```text
Wails
├── Go core
│   ├── capture server
│   ├── archive
│   ├── processing
│   ├── search
│   └── AI providers
└── Svelte UI
    ├── History
    ├── Reader
    ├── AI / Notes
    ├── Follow Browser
    └── Settings
```

The existing `apps/agent` implementation is not discarded. Its services should be reused or moved into a shared Go core so the desktop application does not create a second implementation.

## Follow Browser vs Archive All

These are separate concepts.

### Follow Browser

The browser sends active-page/navigation state. The local Reader follows the page the user is currently viewing.

```text
active browser tab
       ⇅
local Reader
```

### Archive All

Every eligible, deduplicated page is stored in History.

```text
History
├── GitHub repository
├── article
├── Bilibili video + transcript
├── documentation page
└── ...
```

Follow Browser can be enabled while Archive All is disabled, and vice versa.

## Defuddle boundary

Defuddle remains an upstream dependency and owns extraction concerns:

- main content detection;
- consistent cleanup and Markdown conversion;
- metadata and Schema.org extraction;
- site-specific extractors;
- async extractor variables such as transcripts where supported.

This repository owns capture policy, delivery, archive, UI, AI and knowledge behavior.

## Protocol

`ContentPacket` is the durable boundary between browser extraction and local processing.

Stable fields include:

```text
protocolVersion
captureId
capturedAt
source
content
selection
highlights
metadata
media
```

Future browser-mirror fields should be additive within protocol `1.x`, for example navigation/session metadata or active-page state.

## Persistence model

Primary data:

```text
packet.json
source.md
source.html    # planned
raw.html       # optional/planned
assets/        # planned
```

Derived data:

```text
analysis.json
note.md
SQLite/FTS indexes
embeddings
relations
```

Derived data must be rebuildable from the primary archive whenever possible.

Obsidian is not a system dependency. A normal filesystem directory is the default store. Pointing that directory at an Obsidian vault is merely one optional usage pattern.

## AI policy

Automatic capture and automatic AI are intentionally independent.

```text
Auto Capture: ON
Auto AI:      OFF / rules
```

Potential rules:

- paper → analyze automatically;
- video → analyze when transcript exists;
- long article → analyze after dwell/length threshold;
- search/result page → do not analyze.

AI failure never invalidates or removes the page archive.

## Failure model

| Failure | Behavior |
|---|---|
| Defuddle fails | no invalid packet is submitted; diagnostic state is surfaced |
| local app down | extension queues capture |
| auth rejected | queued item is retained |
| disk failure | delivery fails and item remains retryable |
| AI disabled | archive remains fully usable |
| AI fails | primary capture survives; error is inspectable |
| duplicate navigation | fingerprint/dedup prevents archive flood |
| response lost after save | same captureId is idempotent |

## Storage extension point

Filesystem is the default durable sink. Future storage/catalog integrations must stay behind core services.

```go
type Store interface { ... }
```

SQLite catalogs filesystem artifacts; it does not replace them.

## Transport extension point

Core capture/knowledge services must not depend on transport. Input may later arrive from:

- browser HTTP;
- Native Messaging;
- CLI;
- MCP;
- batch import;
- URL/RSS ingestion.

## Reference lessons

### Obsidian Web Clipper

Borrow:

- rendered-page extraction patterns;
- Defuddle integration;
- selection/highlight concepts.

Do not inherit:

- Obsidian-only destination assumptions;
- browser popup as the primary reading surface;
- browser-resident AI as the core architecture.

### Atomic

Borrow:

- local-first processing;
- core vs transport separation;
- durable archive/knowledge thinking;
- future semantic/MCP direction.

Do not inherit database/embedding complexity before automatic capture and the desktop Reader are excellent.
