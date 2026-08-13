# References and upstream boundaries

This project is original glue/application code around upstream components and architectural ideas.

## Defuddle

Repository: `kepano/defuddle`

Role:

- direct npm dependency
- rendered DOM extraction
- metadata/Schema.org extraction
- Markdown conversion through the full bundle
- site-specific extractor variables

No Defuddle source is vendored here.

## Obsidian Web Clipper

Repository: `obsidianmd/obsidian-clipper`

Role: implementation reference for:

- browser extension capture lifecycle
- Defuddle `parseAsync()` integration
- selection/highlight concepts
- variables/templates
- cross-browser product considerations

This repository is not a fork of Obsidian Web Clipper.

## Atomic

Repository: `kenforthewin/atomic`

Role: architectural reference for:

- local-first capture
- queued delivery
- separating core business logic from transport/UI
- future semantic search / MCP direction

Atomic code is not copied into this repository.

## allan-deng/web_clipper

Role: small reference confirming the practical extension → localhost service pattern.

Its Readability/Turndown extraction stack is not used here because Defuddle is the extraction dependency.

## Dependency policy

Prefer using upstream projects through their public package/API boundaries. Fork only when a required change cannot reasonably be contributed upstream or implemented through composition.
