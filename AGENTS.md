# AGENTS.md

This repository is intentionally split by responsibility.

## Non-negotiable architecture rules

1. `apps/extension` owns browser capture only.
2. `apps/agent` owns AI, persistence, local files, future indexing and automation.
3. `packages/protocol/content-packet.schema.json` is the contract between them.
4. Never make the extension write arbitrary local paths.
5. Never make AI success a prerequisite for source persistence.
6. Do not copy Defuddle source into this repository. Consume it as an upstream dependency.
7. Transport handlers must remain thin. Put business logic in services.
8. Derived data (`analysis.json`, `note.md`) must be regenerable from `packet.json`.
9. New protocol fields should be additive within `1.x`; breaking changes require a protocol major bump.
10. Never log API keys, Bearer tokens, full raw HTML, or full AI request payloads.

## Commit convention

Use Conventional Commits:

- `feat(extension): ...`
- `feat(agent): ...`
- `feat(protocol): ...`
- `fix(agent): ...`
- `docs(architecture): ...`
- `test(agent): ...`
- `chore(ci): ...`

Keep changes cohesive. Tests belong in the same commit as the behavior they verify.

## Go

- Standard library first.
- Keep `internal/httpapi` focused on HTTP concerns.
- Validate all external input in `internal/protocol`.
- Use atomic file replacement for derived artifacts.
- Refuse unsafe non-loopback defaults.
- Always close response bodies.
- Add timeouts to outbound HTTP.

## TypeScript extension

- MV3.
- No framework until UI complexity justifies one.
- Content script extracts; background service worker transports/retries.
- Keep API keys out of the extension. Only the local agent token is stored there.
- Treat page DOM/content as hostile input.
- Queue failed captures in `chrome.storage.local`.
- Do not silently truncate source before constructing `ContentPacket`.

## Testing

Before merging:

```bash
cd apps/agent && go test ./...
npm run typecheck
npm run build
```

If dependency installation is unavailable in the execution environment, do not claim the extension build passed; CI is the source of truth.
