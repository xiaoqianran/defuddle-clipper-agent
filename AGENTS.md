# AGENTS.md

This repository is intentionally split by responsibility.

## Product invariant

The browser extension is a **sensor/bridge**. The local desktop application is the **primary product UI**.

Normal use should become:

```text
browse normally
→ page is captured automatically
→ local desktop History/Reader updates
```

Do not optimize the product around a browser popup.

## Non-negotiable architecture rules

1. `apps/extension` owns browser observation, Defuddle extraction, capture coordination and delivery only.
2. The extension must become mostly headless; popup/manual capture is fallback/debug UI.
3. Automatic capture must handle normal navigation and SPA navigation without producing duplicate floods.
4. Reading, history, search, AI, files and future knowledge features belong in the local application/core.
5. `apps/agent` contains the current Go core/localhost service. Future `apps/desktop` must reuse or extract this core rather than duplicate it.
6. `packages/protocol/content-packet.schema.json` is the contract between browser and local core.
7. Never make the extension write arbitrary local paths.
8. Never make AI success a prerequisite for source persistence.
9. Auto Capture and Auto AI are separate capabilities and settings.
10. Do not copy Defuddle source into this repository. Consume it as an upstream dependency.
11. Transport handlers must remain thin. Put business logic in services.
12. Filesystem artifacts are the default durable source. Obsidian is optional and must never become a core dependency.
13. Derived data (`analysis.json`, `note.md`, indexes, embeddings) must be regenerable from primary capture artifacts whenever practical.
14. New protocol fields should be additive within `1.x`; breaking changes require a major protocol bump.
15. Never log API keys, Bearer tokens, full raw HTML, or full AI request payloads.

## Automatic capture rules

The capture coordinator should reason about meaningful page transitions, not arbitrary DOM changes.

Required concepts:

- active tab;
- normal navigation;
- SPA navigation (`pushState`, `replaceState`, `popstate`);
- debounce / DOM stability;
- canonical URL;
- extracted-content fingerprint;
- minimum dwell time where useful;
- pause/resume;
- allowlist/denylist;
- retry queue.

Scrolling, ads, lazy widgets or unrelated DOM mutations must not create repeated captures.

## Desktop application direction

Preferred stack: Wails + Go + Svelte.

The desktop app should provide:

- History;
- large Reader;
- Follow Browser mode;
- Archive All mode;
- AI / Notes panel;
- search;
- capture/AI settings.

The desktop app should own the local server lifecycle once it becomes the default distribution.

## Commit convention

Use Conventional Commits:

- `feat(extension): ...`
- `feat(desktop): ...`
- `feat(agent): ...`
- `feat(protocol): ...`
- `fix(extension): ...`
- `fix(agent): ...`
- `docs(product): ...`
- `docs(architecture): ...`
- `chore(ci): ...`

Keep changes cohesive. Tests belong in the same commit as the behavior they verify.

## Go

- Standard library first unless a dependency materially improves the product.
- Keep HTTP adapters focused on transport concerns.
- Validate external input in protocol/core boundaries.
- Use atomic file replacement for derived artifacts.
- Refuse unsafe non-loopback defaults.
- Always close response bodies.
- Add timeouts to outbound HTTP.
- Keep services reusable by both standalone agent and Wails desktop app.

## TypeScript extension

- MV3.
- No heavy UI framework in the extension.
- Content script observes/extracts; background service worker transports/retries and tracks tab-level state.
- Keep API keys out of the extension. Only the local bridge token may be stored there.
- Treat page DOM/content as hostile input.
- Queue failed captures in `chrome.storage.local`.
- Do not silently truncate source before constructing `ContentPacket`.
- Prefer explicit navigation/fingerprint state machines over broad MutationObserver-driven recapture.

## Testing

Before merging browser/core changes:

```bash
cd apps/agent && go test ./...
npm run typecheck
npm run build
```

Automatic capture work must add tests for navigation dedup/state transitions where practical.

Desktop work must add its own build/test job to CI before it is considered complete.

If dependency installation is unavailable in an execution environment, do not claim a build passed; CI is the source of truth.
