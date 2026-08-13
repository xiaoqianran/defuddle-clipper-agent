# Roadmap

The product goal is: **automatically mirror pages the user browses into a large local desktop application**.

## P0 — Transport foundation

**Status: implemented and CI-verified.**

- [x] MV3 extension
- [x] Defuddle extraction
- [x] ContentPacket v1
- [x] localhost HTTP transport
- [x] retry queue
- [x] raw-first filesystem persistence
- [x] optional OpenAI-compatible analysis
- [x] tests and CI

P0 proves the transport path. It is not the final UX.

## P1 — Automatic browser capture

**Highest priority.**

- [ ] auto-capture normal page loads
- [ ] active-tab tracking
- [ ] SPA navigation detection (`pushState`, `replaceState`, `popstate`)
- [ ] URL-change fallback
- [ ] debounce and DOM-stability coordinator
- [ ] configurable minimum dwell time
- [ ] canonical URL normalization
- [ ] content fingerprint and dedup
- [ ] suppress duplicate captures caused by DOM noise
- [ ] send active-page events for Follow Browser
- [ ] pause/resume Auto Capture
- [ ] domain allowlist / denylist
- [ ] ignore unsupported browser URLs
- [ ] manual `Capture now` remains only as fallback/debug
- [ ] keep persistent retry delivery

Acceptance:

```text
open page A → local receives A automatically
SPA navigation to B → local receives B automatically
scroll / ad refresh / DOM noise → no capture flood
switch active tab → local Follow Browser state changes
```

## P2 — Local desktop application

Preferred stack: **Wails + Go + Svelte**.

- [ ] `apps/desktop` scaffold
- [ ] reuse/embed existing Go core
- [ ] desktop lifecycle owns the local capture endpoint
- [ ] History pane
- [ ] large Reader pane
- [ ] AI / Notes pane
- [ ] Follow Browser mode
- [ ] Archive All mode
- [ ] live current-page state
- [ ] capture status
- [ ] pause Auto Capture from desktop
- [ ] search box
- [ ] render cleaned Markdown/HTML
- [ ] transcript view for video pages
- [ ] capture/data/AI settings
- [ ] desktop build CI

Acceptance:

```text
browse in Chrome/Edge
→ desktop History updates automatically
→ Reader follows active browser page
→ normal use does not require a browser popup
```

## P3 — Durable page archive

- [ ] `source.html`
- [ ] optional `raw.html`
- [ ] image/asset discovery
- [ ] local asset download
- [ ] deterministic asset naming
- [ ] rewrite Markdown/HTML references to local assets
- [ ] capture provenance / extractor diagnostics
- [ ] SQLite catalog
- [ ] migrations
- [ ] full-text search
- [ ] duplicate/collision policy
- [ ] rebuild database from filesystem artifacts

Target capture:

```text
capture/
├── packet.json
├── source.md
├── source.html
├── raw.html          # optional
├── analysis.json     # derived
├── note.md           # derived
└── assets/
```

## P4 — AI understanding

Auto Capture and Auto AI stay independent.

- [ ] processing job state
- [ ] prompt registry/versioning
- [ ] provider/model registry
- [ ] retry/backoff
- [ ] semantic Markdown chunker
- [ ] reprocess existing `packet.json`
- [ ] processor registry: article, documentation, repository, video, paper, discussion, generic
- [ ] stable structured AIResult
- [ ] model/provider/prompt provenance
- [ ] Auto AI rules by page type, dwell time, size and user policy

## P5 — Knowledge layer

- [ ] tags/concepts index
- [ ] related captures
- [ ] embedding provider interface
- [ ] semantic search
- [ ] related-note edges
- [ ] cluster/topic views
- [ ] rebuildable embeddings

## P6 — Agent / automation surface

- [ ] MCP server
- [ ] `capture`, `search`, `read`, `reprocess`
- [ ] CLI
- [ ] WebSocket/SSE events
- [ ] external automation API

## P7 — Capture ecosystem

- [ ] Firefox
- [ ] Safari investigation
- [ ] Native Messaging
- [ ] optional SingleFile adapter
- [ ] optional yt-dlp fallback
- [ ] RSS/URL ingestion
- [ ] batch import

## Explicit non-goals

The project is not an Obsidian-specific app, browser-popup reader, cloud SaaS, general crawler, or vector database-first product.

Core product:

```text
browser activity
→ automatic local copy
→ large desktop reader/history
→ optional AI
→ durable personal web archive
```
