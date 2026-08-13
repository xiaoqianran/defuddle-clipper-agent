# Architecture

## System boundary

```text
┌──────────────────────── Browser ─────────────────────────┐
│                                                         │
│  DOM                                                    │
│   │                                                     │
│   ▼                                                     │
│  Defuddle                                               │
│   │ cleaned content + metadata + site variables         │
│   ▼                                                     │
│  ContentPacket v1                                       │
│   │                                                     │
│   ├─ popup feedback                                     │
│   └─ background delivery + retry queue                  │
└─────────────────────────┬───────────────────────────────┘
                          │ HTTP JSON
                          ▼
┌──────────────────── Local Agent ────────────────────────┐
│                                                        │
│  HTTP adapter                                          │
│      │                                                 │
│      ▼                                                 │
│  CaptureService                                        │
│      │                                                 │
│      ├──► Store packet.json + source.md FIRST          │
│      │                                                 │
│      ├──► Analyzer                                     │
│      │      ├─ short document → one analysis           │
│      │      └─ long document → chunks → synthesis      │
│      │                                                 │
│      └──► Renderer → note.md                           │
│                                                        │
└────────────────────────────────────────────────────────┘
```

## Why the browser is thin

Browser extensions are good at:

- seeing the rendered authenticated DOM
- user interaction
- selections/highlights
- handing captured data to another process

They are a poor place for:

- filesystem orchestration
- long-running jobs
- model credentials
- embeddings/indexes
- durable queues
- subprocesses
- large local databases

Therefore the extension never becomes the knowledge engine.

## Why Defuddle is upstream

Defuddle already owns the hardest generic extraction concerns:

- main content detection
- consistent HTML/Markdown cleanup
- metadata and Schema.org extraction
- site-specific extractors
- async variables such as transcripts where supported

This repository defines policy around extraction; it does not fork the extraction engine.

## Protocol

`ContentPacket` is deliberately richer than a single Markdown string.

The stable fields are:

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

`content.markdown` is the primary AI context in P0.

`content.html` is optional because raw HTML:

- is larger
- is less useful to most text models
- increases privacy exposure
- should not be required for downstream processing

`metadata.variables` preserves Defuddle extractor-specific variables without coupling the protocol to every supported website.

## Persistence model

The raw capture is primary data:

```text
packet.json
source.md
```

Everything below is derived:

```text
analysis.json
analysis-error.txt
note.md
```

This gives the project a useful invariant:

> Changing the model, prompt, renderer, chunker or knowledge schema never requires revisiting the original webpage if the packet contains sufficient source material.

P1 may add a separate raw HTML/SingleFile archival artifact for stronger source fidelity.

## AI pipeline

### Short content

```text
markdown
  ↓
structured prompt
  ↓
Analysis
```

### Long content

```text
markdown
  ↓
heading/paragraph chunks
  ↓
partial Analysis × N
  ↓
JSON aggregation
  ↓
final synthesis
  ↓
Analysis
```

The output schema is stable even when providers/models change.

## Failure model

| Failure | Behavior |
|---|---|
| Defuddle fails | no packet is submitted; extension shows extraction error |
| local agent down | extension queues packet |
| auth rejected | packet remains queued |
| disk failure | request fails; extension queues packet |
| AI disabled | packet saved; note generated without AI |
| AI fails | packet saved; error artifact written; note generated without AI |
| response lost after successful save | retry uses same captureId; agent returns duplicate |

## Future extension points

### Storage

```go
type Store interface { ... }
```

P0 ships filesystem storage. SQLite in P3 should catalog, not replace, the raw artifacts.

### AI

```go
type Analyzer interface {
    Analyze(context.Context, protocol.ContentPacket) (Analysis, error)
}
```

Provider-specific HTTP belongs behind this interface.

### Transport

The capture service does not know whether input came from:

- HTTP
- Native Messaging
- CLI
- MCP
- batch import

That separation is intentional.

## Reference architecture lessons

### Obsidian Web Clipper

Borrow:

- rendered-page capture model
- Defuddle integration
- variable mindset
- selection/highlight UX concepts

Do not inherit:

- Obsidian-only output assumptions
- browser-resident AI as the primary architecture

### Atomic

Borrow:

- local-first processing
- core vs transport separation
- queued capture mindset
- future semantic/MCP direction

Do not inherit yet:

- the full application stack
- database/embedding complexity before capture is stable
