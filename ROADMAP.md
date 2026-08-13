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

## P1 — Automatic browser capture

**Status: implemented and CI-verified.**

- [x] auto-capture normal page loads
- [x] active-tab tracking
- [x] SPA/history navigation detection
- [x] URL-change fallback
- [x] capture delay and DOM-stability coordinator
- [ ] true active dwell-time policy
- [x] canonical URL normalization
- [x] content fingerprint and dedup
- [x] suppress duplicate captures caused by DOM noise
- [x] send active-page events for Follow Browser
- [x] pause/resume Auto Capture
- [x] domain allowlist / denylist
- [x] ignore unsupported browser URLs
- [x] manual `Capture now` only as fallback/debug
- [x] persistent retry delivery

Acceptance achieved:

```text
open page A → local receives A automatically
SPA navigation to B → local receives B automatically
scroll / ad refresh / DOM noise → no capture flood
switch active tab → local Follow Browser state changes
```

## P2 — Local desktop application

**Status: first usable reader implemented.** Stack: **Wails + Go + Svelte**.

- [x] `apps/desktop` scaffold
- [x] local-only Go client for the existing agent
- [x] filesystem-backed History API
- [x] single-capture Reader API
- [x] History pane
- [x] large Reader pane
- [x] AI / Notes pane
- [x] Follow Browser mode
- [x] live current-page state
- [x] search box
- [x] transcript view for video pages
- [x] frontend typecheck/build included in CI
- [ ] embed/reuse the agent runtime inside the desktop process
- [ ] desktop lifecycle owns the local capture endpoint
- [ ] Archive All control in desktop UI
- [ ] detailed capture/queue status
- [ ] pause Auto Capture from desktop
- [ ] rendered Markdown/cleaned HTML reader mode
- [ ] capture/data/AI settings UI
- [ ] native Wails packaging CI

Current acceptance:

```text
browse in Chrome/Edge
→ desktop History updates automatically
→ Reader follows active browser page
→ manual History selection exits Follow Browser
→ normal reading does not happen inside the browser popup
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
