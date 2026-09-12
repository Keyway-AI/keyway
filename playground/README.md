# Keyway token inspector — browser playground

A static, client-side page that runs Keyway's agent-auth analyzer
(`internal/agentauth`, the *same* code the CLI uses) compiled to WebAssembly. Paste
an agent / MCP / OAuth bearer token and see the findings — **the token never leaves
the browser**, so it is safe to share as a public tool.

## Build

```bash
make playground        # -> playground/keyway.wasm + playground/wasm_exec.js
```

`keyway.wasm` is a build artifact (gitignored); everything else is committed.

## Run locally

WebAssembly must be served over HTTP (not `file://`):

```bash
cd playground && python3 -m http.server 8090
# open http://localhost:8090
```

## Deploy

The `playground/` directory is a plain static site (`index.html`, `wasm_exec.js`,
`keyway.wasm`). After `make playground`, deploy the folder to any static host —
e.g. Netlify drag-and-drop, GitHub Pages, or:

```bash
npx vercel deploy playground --prod
```

Rebuild `keyway.wasm` (`make playground`) whenever the analyzer changes so the page
stays in sync with the CLI.

## Files

| File | What |
|---|---|
| `main.go` | WASM entrypoint (`//go:build js && wasm`) — exposes `window.keywayInspect(token, opts)` |
| `stub.go` | non-wasm stub so `go build ./...` stays green |
| `index.html` | the page (self-contained; no external dependencies) |
| `wasm_exec.js` | Go's WASM runtime glue (copied from the toolchain) |
