# Defuddle Clipper Agent

A local-first **browser mirror and web inbox**.

The browser extension is intentionally tiny and mostly headless. The real product is a large local desktop application that automatically receives, archives, reads, and optionally analyzes the pages you visit.

## Product goal

```text
You browse normally in Chrome / Edge
              ↓
      page / SPA navigation
              ↓
      headless MV3 extension
              ↓
          Defuddle
              ↓
       ContentPacket v1
              ↓
        localhost HTTP
              ↓
┌────────────────────────────────────────────┐
│              Local Desktop App             │
│                                            │
│  History        Reader          AI / Notes │
│  GitHub         full page       summary    │
│  Zhihu          Markdown/HTML   key points │
│  Bilibili       transcript      concepts   │
│  arXiv          code/tables     questions  │
└────────────────────────────────────────────┘
              ↓
        local durable archive
```

The intended interaction is **not** `open page → click a tiny browser popup → read inside the extension`.

It is:

```text
open page → page is copied automatically → local app follows the browser
```

## Product principles

1. **Auto-capture first.** Normal browsing should be enough; manual capture is only a fallback.
2. **Browser is a sensor, desktop is the product.** Reading, history, AI, files, search and configuration live locally.
3. **Local-first and destination-neutral.** Obsidian is optional. Plain local files are the default durable format.
4. **Capture everything, analyze selectively.** Auto Capture can be always-on while Auto AI remains off or rule-driven.
5. **Raw source before enrichment.** AI/provider failures must never lose a captured page.
6. **Defuddle is upstream.** Use its generic and site-specific extractors instead of rebuilding extraction.
7. **Stable protocol boundary.** Browser, desktop UI, storage and AI providers remain replaceable.

## Current state: P0 transport foundation

Already implemented and CI-verified:

- Chromium MV3 extension written in TypeScript.
- Defuddle extraction and Markdown conversion.
- Metadata, Schema.org and extractor variables, including transcript when Defuddle provides it.
- Versioned `ContentPacket` v1 contract.
- `localhost` delivery with optional Bearer token.
- Extension-side persistent retry queue when the local agent is unavailable.
- Go local agent with no third-party runtime dependencies.
- Raw `packet.json` and `source.md` persisted before AI.
- OpenAI-compatible optional analysis.
- Long-document chunk + synthesis.
- Idempotency by `captureId`.
- Security checks for loopback binding and capture paths.
- Go tests and GitHub Actions CI.

P0 proves transport and persistence. It still uses an explicit capture action and does **not yet** implement the final interaction model.

## Next milestone: automatic browser mirror

```text
Browser navigation
  ↓
URL / SPA change detection
  ↓
debounce + DOM stability
  ↓
Defuddle extraction
  ↓
content fingerprint / dedup
  ↓
automatic local delivery
  ↓
Desktop History + Reader follows current browser page
```

Required behaviors:

- capture normal page loads automatically;
- detect SPA navigation (`pushState`, `replaceState`, `popstate`, URL changes);
- avoid repeated captures caused by ads, scrolling or incremental DOM mutations;
- configurable debounce / minimum dwell time;
- pause switch, allowlist and denylist;
- ignore browser-internal/private pages by default;
- manual **Capture now** remains available only as a fallback;
- local desktop UI becomes the primary user-facing interface.

See [`ROADMAP.md`](ROADMAP.md).

## Target repository layout

```text
.
├── apps/
│   ├── extension/          # headless browser sensor / capture bridge
│   ├── agent/              # Go core + localhost service (current P0)
│   └── desktop/            # Wails + Svelte local UI (next)
├── packages/
│   └── protocol/           # versioned ContentPacket contract
├── docs/
│   ├── ARCHITECTURE.md
│   ├── SECURITY.md
│   └── REFERENCES.md
├── .github/workflows/
├── AGENTS.md
└── ROADMAP.md
```

The current standalone agent is kept because its core services are useful. The desktop application should progressively embed/reuse that Go core rather than create a second implementation.

## Local archive model

Obsidian is **not required**. `DCA_DATA_DIR` can point to any normal folder.

```text
<DCA_DATA_DIR>/
└── captures/
    └── YYYY/MM/DD/<capture-id>/
        ├── packet.json       # canonical structured capture
        ├── source.md         # cleaned text for reading/search/AI
        ├── source.html       # planned: cleaned HTML for local reader
        ├── raw.html          # planned/optional: stronger source fidelity
        ├── analysis.json     # optional derived AI result
        ├── note.md           # optional rendered derivative
        └── assets/           # planned localized images/assets
```

Filesystem artifacts remain portable and inspectable. Future SQLite/FTS/embedding indexes are derived catalogs and must be rebuildable.

## Run the current P0 agent

Requirements: Go 1.22+.

```bash
cd apps/agent

export DCA_DATA_DIR="$HOME/dca-data"
export DCA_TOKEN="replace-with-a-long-random-token"

go run ./cmd/clipper-agent
```

Default address: `127.0.0.1:27123`.

Health check:

```bash
curl http://127.0.0.1:27123/health
```

### Optional AI

AI is disabled by default and remains independent from automatic capture.

```bash
export DCA_AI_ENABLED=true
export DCA_OPENAI_BASE_URL="https://api.example.com/v1"
export DCA_OPENAI_API_KEY="..."
export DCA_OPENAI_MODEL="your-model-id"
```

Long term, Auto AI is rule-driven, for example:

```text
paper       → always
repository  → optional
video       → if transcript exists
article     → if dwell time / length threshold is met
search page → never
```

## Build the current extension

Requirements: Node.js 20+.

```bash
npm install
npm run build
```

Load `apps/extension/dist` from `chrome://extensions` → **Developer mode** → **Load unpacked**.

Current P0 still exposes manual capture. This UI is transitional; the target extension is headless except for status/settings/pause controls.

## Desktop application direction

Preferred stack:

```text
Wails
├── Go core
│   ├── localhost capture server
│   ├── archive services
│   ├── processing jobs
│   ├── search/indexing
│   └── AI providers
└── Svelte UI
    ├── History
    ├── Reader
    ├── AI / Notes
    ├── Follow Browser
    └── Settings
```

Primary UI concept:

```text
┌──────────────────────────────────────────────────────────────┐
│ Search                     Auto Capture ●   Auto AI ○       │
├──────────────┬────────────────────────────┬──────────────────┤
│ HISTORY      │ READER                     │ AI / NOTES       │
│ GitHub       │ cleaned page / Markdown    │ summary          │
│ Zhihu        │ images / code / tables     │ key points       │
│ Bilibili     │ transcript                 │ concepts         │
│ arXiv        │                            │ questions        │
├──────────────┴────────────────────────────┴──────────────────┤
│ current source URL · capture time · extractor · status      │
└──────────────────────────────────────────────────────────────┘
```

Two core modes:

- **Follow Browser** — local Reader switches to the page currently active in the browser.
- **Archive All** — every eligible page becomes part of the searchable local history.

## Explicit non-goals

The project is not:

- an Obsidian plugin or Obsidian clone;
- a browser-popup reading application;
- a cloud SaaS;
- a general crawler that indiscriminately fetches the public web;
- a vector database with a capture feature bolted on.

It is a **local web inbox / browser mirror**: automatically copy what the user actually browses, preserve it locally, present it in a large desktop reader, and add AI/knowledge capabilities on top.

## Development

```bash
make test
make build
```

Architecture rules are in [`AGENTS.md`](AGENTS.md).

## License

MIT. See [`LICENSE`](LICENSE).
