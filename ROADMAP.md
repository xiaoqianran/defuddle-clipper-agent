# Roadmap

The roadmap is capability-driven. A phase is complete only when the feature is usable end-to-end and tested.

## P0 — Durable capture pipeline

**Status: implemented in initial repository scaffold.**

- [x] MV3 extension
- [x] Defuddle extraction
- [x] Markdown conversion
- [x] extractor variables / transcript forwarding
- [x] ContentPacket v1 schema
- [x] localhost HTTP transport
- [x] optional Bearer auth
- [x] extension offline/retry queue
- [x] raw-first filesystem persistence
- [x] OpenAI-compatible provider
- [x] long-document chunk + synthesis
- [x] Markdown rendering
- [x] idempotency by captureId
- [x] Go tests
- [x] CI

Acceptance:

```text
page → extension → Defuddle → ContentPacket → local agent
     → packet.json/source.md → optional AI → note.md
```

An unavailable AI provider must not lose the capture.

## P1 — Rich capture

- [ ] selection-only capture mode
- [ ] highlights and annotations
- [ ] page screenshot reference
- [ ] image discovery
- [ ] local image downloader
- [ ] deterministic asset names
- [ ] HTML snapshot option
- [ ] capture provenance and extractor diagnostics

## P2 — Processing quality

- [ ] semantic Markdown chunker
- [ ] page-type-specific analysis schemas
- [ ] prompt registry
- [ ] model/provider registry
- [ ] retry/backoff
- [ ] processing job state
- [ ] reprocess an existing packet
- [ ] prompt/model provenance stored with analysis
- [ ] configurable output templates

## P3 — Local knowledge store

- [ ] SQLite catalog
- [ ] migrations
- [ ] canonical URL normalization
- [ ] duplicate detection
- [ ] full-text search
- [ ] tags/concepts index
- [ ] related captures
- [ ] import existing Markdown

## P4 — Semantic knowledge

- [ ] embedding provider interface
- [ ] vector storage
- [ ] semantic search
- [ ] related-note edges
- [ ] cluster/topic views
- [ ] incremental re-embedding

## P5 — Agent surface

- [ ] MCP server
- [ ] `capture`, `search`, `read`, `reprocess` tools
- [ ] CLI
- [ ] WebSocket/SSE processing events
- [ ] external automation API
- [ ] health/status diagnostics

## P6 — Capture ecosystem

- [ ] Firefox packaging
- [ ] Safari investigation
- [ ] Native Messaging transport
- [ ] optional SingleFile archive adapter
- [ ] optional yt-dlp fallback for unsupported transcript/video cases
- [ ] RSS/URL ingestion without browser
- [ ] batch capture

## Explicit non-goals

Until P3/P4, do not turn the project into:

- a general-purpose note editor
- an Obsidian clone
- a cloud SaaS
- a crawler platform
- a vector database

The project remains a capture/process/knowledge bridge.
