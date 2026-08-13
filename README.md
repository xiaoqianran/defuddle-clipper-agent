# Defuddle Clipper Agent

A local-first **browser mirror and web inbox**.

The browser extension is intentionally small and mostly headless. The primary UI is a large local desktop reader that follows what you browse, keeps a durable history, and can show optional AI-derived notes.

```text
Chrome / Edge
    ↓ normal page load / SPA navigation / active-tab switch
MV3 browser sensor
    ↓
Defuddle
    ↓
ContentPacket v1
    ↓ localhost
Go local agent
    ├─ durable archive
    ├─ current browser state
    ├─ History / Reader API
    └─ optional AI
    ↓
Wails + Svelte desktop app
    ├─ History
    ├─ large Reader
    ├─ Follow Browser
    ├─ Transcript
    └─ AI / Notes
```

The intended interaction is:

```text
open a page in the browser
→ it is copied automatically
→ it appears in local History
→ the large Reader follows the active browser page
```

There is no Obsidian requirement. `DCA_DATA_DIR` can be any normal local folder.

## Current status

### P0 — transport and persistence ✅

- Defuddle extraction and Markdown conversion
- ContentPacket v1
- localhost HTTP transport
- persistent extension retry queue
- raw-first filesystem persistence
- optional OpenAI-compatible AI
- long-document chunk + synthesis
- tests and CI

### P1 — automatic browser mirror ✅

- normal page loads are captured automatically
- History API / SPA navigation is observed through `webNavigation`
- active-tab/window focus state is reported for Follow Browser
- capture delay + DOM-stability check
- canonical URL normalization
- content fingerprint deduplication
- duplicate suppression for noisy page updates
- pause/resume Auto Capture
- domain allowlist / denylist
- unsupported browser URLs ignored
- manual **Capture now** kept only as fallback/debug

### P2 — first desktop Reader ✅ / partial

Implemented:

- `apps/desktop` Wails + Svelte application
- 1500×950 large desktop window
- filesystem-backed History API
- single-capture Reader API
- History/search pane
- large Reader pane
- AI / Notes pane
- Follow Browser mode
- transcript view when Defuddle provides transcript data
- live active-browser state
- desktop frontend included in root npm typecheck/build CI

Still planned inside P2:

- embed the Go agent lifecycle into the desktop process
- rendered Markdown / cleaned HTML mode
- desktop controls for Auto Capture and queue state
- desktop settings for archive and AI
- native Wails packaging CI

For now the desktop app connects to the standalone local agent. This keeps the UI/data boundary stable while the embedded runtime is refactored.

## Repository layout

```text
.
├── apps/
│   ├── extension/          # automatic browser sensor
│   ├── agent/              # Go localhost service + archive/AI
│   └── desktop/            # Wails + Svelte large local reader
├── packages/
│   └── protocol/           # ContentPacket v1 JSON schema
├── docs/
│   ├── ARCHITECTURE.md
│   ├── SECURITY.md
│   └── REFERENCES.md
├── AGENTS.md
└── ROADMAP.md
```

## Quick start — Windows / PowerShell

### 1. Build the extension and desktop frontend

Requirements: Node.js 20+.

```powershell
git clone https://github.com/xiaoqianran/defuddle-clipper-agent.git
cd defuddle-clipper-agent

npm install
npm run typecheck
npm run build
```

`npm run build` builds both:

```text
apps/extension/dist
apps/desktop/frontend/dist
```

### 2. Start the local agent

Requirements: Go 1.22+.

Open a PowerShell window:

```powershell
cd apps/agent

$env:DCA_DATA_DIR="$HOME\dca-data"
$env:DCA_TOKEN="replace-with-a-long-random-token"

go run ./cmd/clipper-agent
```

Default endpoint:

```text
http://127.0.0.1:27123
```

Health check from another PowerShell:

```powershell
Invoke-RestMethod http://127.0.0.1:27123/health
```

### 3. Load the extension

Open:

```text
chrome://extensions
```

Then:

```text
Developer mode
→ Load unpacked
→ select apps/extension/dist
```

Open extension settings and use:

```text
Agent URL: http://127.0.0.1:27123
Token:     same value as DCA_TOKEN
Auto Capture: ON
Follow Browser: ON
```

After this, normal browsing is enough. You do not need to press **Capture now**.

### 4. Start the large desktop Reader

Open another PowerShell from the repository:

```powershell
cd apps/desktop

$env:DCA_TOKEN="replace-with-a-long-random-token"

go mod tidy
go run .
```

The desktop window reads from the same local agent.

For Wails hot-reload development, install the pinned compatible CLI:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails dev
```

The repository pins Wails v2.12.0 because the existing project uses Go 1.22; the newer Wails v2.13.0 module requires Go 1.25.

## Desktop behavior

### Follow Browser

When enabled:

```text
active browser tab changes
→ extension reports current URL/title
→ desktop polls local browser state
→ matching mirrored capture opens in Reader
```

If the page is still being extracted, the desktop waits until it appears in History.

### Inspect old history

Clicking an older History item intentionally disables Follow Browser, so the Reader does not immediately jump back to the current browser tab.

Re-enable **Follow Browser** when you want the Reader to track the browser again.

### Transcript

When Defuddle exposes a video transcript through extractor variables, the capture carries it in:

```text
ContentPacket.media.transcript
```

The desktop Reader then exposes a **Transcript** tab.

## Local archive

```text
<DCA_DATA_DIR>/
└── captures/
    └── YYYY/MM/DD/<capture-id>/
        ├── packet.json
        ├── source.md
        ├── analysis.json       # optional AI derivative
        ├── analysis-error.txt  # if AI fails
        └── note.md
```

`packet.json` and `source.md` are written before AI runs. AI failure therefore does not lose the captured page.

Planned P3 additions:

```text
source.html
raw.html
assets/
SQLite catalog
full-text search
```

The filesystem remains primary data; future indexes must be rebuildable.

## Local APIs

Current important endpoints:

```text
GET  /health
POST /v1/captures
GET  /v1/captures?limit=100
GET  /v1/captures/{captureId}
POST /v1/captures/{captureId}/reprocess
POST /v1/browser/active
GET  /v1/browser/state
```

The desktop uses these APIs through its Go bridge; the Svelte UI never receives the Bearer token directly.

## Optional AI

Automatic capture and AI are independent. AI is disabled by default.

Any OpenAI-compatible `/chat/completions` endpoint works (OpenAI, NVIDIA NIM, vLLM, local servers).

```powershell
$env:DCA_AI_ENABLED="true"
$env:DCA_OPENAI_BASE_URL="https://api.example.com/v1"
$env:DCA_OPENAI_API_KEY="..."
$env:DCA_OPENAI_MODEL="your-model-id"
```

NVIDIA NIM example. This model is multimodal **text-out**: it can read a cover image with the Markdown, but it does not generate images (`/v1/images/generations` is not used).

```powershell
$env:DCA_AI_ENABLED="true"
$env:DCA_OPENAI_BASE_URL="https://integrate.api.nvidia.com/v1"
$env:DCA_OPENAI_API_KEY=""
$env:DCA_OPENAI_MODEL="google/diffusiongemma-26b-a4b-it"
$env:DCA_AI_TIMEOUT_SECONDS="180"
```

When `ContentPacket.metadata.image` is an `http(s)` URL, the agent forwards it as an OpenAI-style `image_url` part. Data URIs and other non-http schemes are skipped. `analysis.json` records model, provider host, prompt version, whether an image was sent, and the analysis timestamp. API keys are never written to stored files.

A captured source is still saved if the provider is down.

## Design rules

1. **Browser is a sensor; desktop is the product.**
2. **Capture first, enrich second.**
3. **Plain files are the default durable format.**
4. **Obsidian is optional, never a dependency.**
5. **Auto Capture and Auto AI are separate policies.**
6. **Defuddle stays upstream; do not fork extraction into this repo.**
7. **ContentPacket is the stable browser/local boundary.**
8. **Derived AI/search artifacts must be rebuildable.**

See [`ROADMAP.md`](ROADMAP.md) for the remaining P2 work and P3–P7 plan.

## License

MIT. See [`LICENSE`](LICENSE).
