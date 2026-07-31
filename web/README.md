# Keyway Web

The Keyway dashboard — React + TypeScript + Vite + Tailwind v4 — over the HTTP
API (PRD §12).

## Develop

```bash
npm install
npm run dev            # http://localhost:5173, proxies /v1 -> :8080
```

Point at a running backend:

```bash
# terminal 1
make serve             # keyway API on :8080
# terminal 2
npm run dev
```

Set a bearer token in the browser console (matches `KEYWAY_API_TOKEN`):

```js
localStorage.setItem("keyway.token", "your-token");
```

### Mock mode

Most endpoints return `501 Not Implemented` until their milestone lands
(see [`../docs/progress.md`](../docs/progress.md)). The client transparently falls back to
sample data (`src/api/mock.ts`) so the whole UI is navigable. Force live-only:

```js
localStorage.setItem("keyway.live", "1");
```

## Build

```bash
npm run build          # tsc -b && vite build -> dist/
```

`make docker` embeds `dist/` into the Go binary so `keyway serve` serves the UI.

## Structure

```
src/
├── api/        typed client, response types, mock data
├── components/ Layout + shared UI primitives
├── lib/        formatting, data-fetch hook, blast-radius resolver
└── pages/      Dashboard, Consumers, Changes, Blast radius, Canary
```
