# Security

A web clipper bridges hostile web content and local files. Treat this boundary as security-sensitive.

## Threat model

Inputs include:

- arbitrary DOM text and metadata
- arbitrary URLs
- arbitrary Defuddle extractor variables
- forged requests to localhost
- malicious model output
- malformed capture IDs
- oversized request bodies

## P0 controls

### Loopback by default

The server binds to:

```text
127.0.0.1:27123
```

If configured to bind to a non-loopback host, startup fails unless `DCA_TOKEN` is set.

### Optional Bearer authentication

Set a long random `DCA_TOKEN`.

The extension sends:

```text
Authorization: Bearer <token>
```

Do not reuse a cloud API key as this token.

### JSON-only endpoint

`POST /v1/captures` requires JSON. The server does not add permissive CORS headers.

### Bounded request body

`DCA_MAX_BODY_BYTES` defaults to 10 MiB.

### Path validation

`captureId` is restricted to a short safe character set and is never treated as an arbitrary path.

### Atomic derived writes

Files are written through temporary files and renamed into place to avoid partially written artifacts.

### Raw-first processing

An external model cannot prevent the source packet from being saved after the request is accepted.

### AI credentials stay local

Model/API credentials are environment variables of the local agent. They are never stored by the extension.

## Remaining risks

P0 does not:

- encrypt the capture directory
- sandbox model output
- localize and inspect page assets
- archive a cryptographically exact source page
- authenticate individual browser extension IDs
- provide OS-level process isolation

Do not expose the HTTP server directly to the Internet.

## Reporting

For security issues, avoid posting secrets or exploit payloads containing private captured content in a public issue.
