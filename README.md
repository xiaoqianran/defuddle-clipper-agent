# Defuddle Clipper Agent

A local-first web capture and AI knowledge pipeline.

**Goal:** turn the current browser page into a durable, structured local knowledge artifact with the smallest possible browser-side surface.

```text
Browser page
   ↓
Defuddle
   ↓
ContentPacket v1
   ↓
MV3 extension
   ↓
localhost HTTP
   ↓
Go agent
   ├─ raw packet persistence
   ├─ OpenAI-compatible analysis
   ├─ long-document chunk + synthesis
   └─ Markdown rendering
   ↓
local knowledge directory / Obsidian vault
```

## Why this repository exists

This project deliberately does **not** fork Obsidian Web Clipper or Defuddle.

- [`defuddle`](https://github.com/kepano/defuddle) is used as the extraction dependency.
- Obsidian Web Clipper is a reference for browser-side UX, selection/highlight flows, and extraction integration.
- Atomic is a reference for local-first knowledge processing, offline delivery, and future semantic/MCP layers.

The core boundary is the versioned `ContentPacket` protocol. The extension can be replaced without rewriting the agent, and the agent can evolve without coupling the browser to storage or AI providers.

## Current milestone: P0 end-to-end capture

Implemented:

- Chromium MV3 extension written in TypeScript.
- Defuddle-based page extraction and Markdown conversion.
- Metadata + Schema.org + extractor variables (including transcript when Defuddle provides it).
- Versioned `ContentPacket` JSON contract.
- `localhost` delivery with optional Bearer token.
- Persistent extension-side retry queue for an unavailable local agent.
- Go local agent with no third-party runtime dependencies.
- Raw packet saved **before** AI processing.
- OpenAI-compatible AI provider.
- Section/paragraph-aware long-document chunking and hierarchical synthesis.
- Markdown note rendering.
- Idempotency by `captureId`.
- Security checks for non-loopback binding and capture paths.
- Go tests and GitHub Actions CI.

Not in P0:

- image localization
- browser highlights/annotations
- SQLite/full-text search
- embeddings/semantic relations
- MCP server
- Native Messaging
- Firefox/Safari packaging

These are intentionally staged in [`ROADMAP.md`](ROADMAP.md).

## Repository layout

```text
.
├── apps/
│   ├── extension/          # Chromium MV3 capture client
│   └── agent/              # local Go service
├── packages/
│   └── protocol/           # JSON Schema for ContentPacket
├── docs/
│   ├── ARCHITECTURE.md
│   ├── SECURITY.md
│   └── REFERENCES.md
├── .github/workflows/
├── AGENTS.md
├── Makefile
└── ROADMAP.md
```

## 1. Run the local agent

Requirements: Go 1.22+.

```bash
cd apps/agent

export DCA_DATA_DIR="$HOME/ObsidianVault/Inbox/WebClips"
export DCA_TOKEN="replace-with-a-long-random-token"

go run ./cmd/clipper-agent
```

Default listen address:

```text
127.0.0.1:27123
```

Health check:

```bash
curl http://127.0.0.1:27123/health
```

### Enable AI

AI is disabled by default. The provider is OpenAI Chat Completions compatible.

```bash
export DCA_AI_ENABLED=true
export DCA_OPENAI_BASE_URL="https://api.example.com/v1"
export DCA_OPENAI_API_KEY="..."
export DCA_OPENAI_MODEL="your-model-id"
```

Local compatible endpoints can leave the API key empty if the server does not require one.

## 2. Build the extension

Requirements: Node.js 20+.

```bash
npm install
npm run build
```

Load:

```text
apps/extension/dist
```

from `chrome://extensions` → **Developer mode** → **Load unpacked**.

Open extension settings and set:

```text
Agent URL: http://127.0.0.1:27123
Token:     same value as DCA_TOKEN
```

Then open any page and click **Capture page**.

## 3. Output

Each capture is immutable at the protocol boundary and stored by date:

```text
<DCA_DATA_DIR>/
└── captures/
    └── 2026/
        └── 08/
            └── 13/
                └── <capture-id>/
                    ├── packet.json
                    ├── source.md
                    ├── analysis.json       # when AI succeeds
                    ├── analysis-error.txt  # when AI fails
                    └── note.md
```

`packet.json` and `source.md` are persisted before AI is invoked, so an AI outage never loses the captured source.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `DCA_ADDR` | `127.0.0.1:27123` | listen address |
| `DCA_DATA_DIR` | `~/.defuddle-clipper-agent` | capture root |
| `DCA_TOKEN` | empty | optional Bearer token; strongly recommended |
| `DCA_MAX_BODY_BYTES` | `10485760` | max capture payload |
| `DCA_AI_ENABLED` | `false` | enable AI analysis |
| `DCA_OPENAI_BASE_URL` | `https://api.openai.com/v1` | compatible API base |
| `DCA_OPENAI_API_KEY` | empty | provider key |
| `DCA_OPENAI_MODEL` | empty | required when AI is enabled |
| `DCA_AI_CHUNK_CHARS` | `12000` | approximate max chars per analysis chunk |
| `DCA_AI_TIMEOUT_SECONDS` | `90` | per-request timeout |

If `DCA_ADDR` is not loopback, the agent refuses to start unless `DCA_TOKEN` is set.

## Development

```bash
make test
make build
```

Architecture rules and repository conventions are in [`AGENTS.md`](AGENTS.md).

## Design principles

1. **Capture first, enrich second.** Raw source must survive AI/provider failures.
2. **Thin browser, capable local agent.** Files, queues, AI and future search stay outside the extension.
3. **Stable protocol boundary.** Transport and storage are replaceable.
4. **Defuddle, do not reinvent extraction.**
5. **Local-first by default.** No cloud component is required.
6. **Derived artifacts are reproducible.** A future model/prompt can regenerate `analysis.json` and `note.md` from `packet.json`.

## License

MIT. See [`LICENSE`](LICENSE).
