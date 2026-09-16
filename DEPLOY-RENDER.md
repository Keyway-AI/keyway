# Deploying the Keyway backend on Render

The backend is a long-running Go daemon (`keyway serve`) that needs Postgres —
so it goes on Render (a container + database host), **not** Vercel (static +
serverless, which hosts the frontend). This repo ships a Render Blueprint
([`render.yaml`](render.yaml)) that provisions everything.

## What you get

One web service (the API **and** the web UI, same origin) plus a managed Postgres.
The single binary:

- **self-migrates** on start (`KEYWAY_MIGRATE`), so there is no separate migrate
  step (the image is distroless and has no shell);
- **self-seeds** — the scheduler discovers the real example configs baked into the
  image (`/seed/manifests`, from `bench/oss/manifests`) and persists a genuine
  baseline, so the dashboards come up on **real** data, not fixtures.

## Deploy (about 5 minutes)

1. Make sure `render.yaml` and the updated `Dockerfile` are on `main` (this PR).
2. Go to **[dashboard.render.com](https://dashboard.render.com) → New → Blueprint**.
3. **Connect** the `Keyway-AI/keyway` repo (authorize Render for GitHub if asked).
4. Render reads `render.yaml` and shows a plan: a web service **keyway-api** and a
   Postgres **keyway-db**. Click **Apply**.
5. Wait for the build + first deploy. When it's live you'll get a URL like
   `https://keyway-api-XXXX.onrender.com`.

## Verify it's real

```bash
API=https://keyway-api-XXXX.onrender.com
curl -s $API/v1/health                       # {"status":"ok",...}
# grab the token Render generated:
#   keyway-api → Environment → KEYWAY_API_TOKEN → copy
curl -s $API/v1/consumers -H "Authorization: Bearer <KEYWAY_API_TOKEN>" | jq '.consumers[].name'
#   -> httpbin, ingressgateway, graphql, request-authentication, provider_name1
```

## Use the live UI

The deployed URL serves the web app too. Open it, go to **Settings**, enable the
live API, and paste the **KEYWAY_API_TOKEN** (from the service's Environment tab).
The dashboards now read from the live backend (real inventory, real coverage, real
snapshots) — and drift/probe history fill in over time as the scheduler runs.

To instead point the **Vercel** frontend at this backend, set
`VITE_CLOUD_API_URL=https://keyway-api-XXXX.onrender.com` in the Vercel project and
redeploy. (The Vercel site otherwise stays the fast, no-backend demo on the baked-in
real example dataset.)

## Notes & costs

- **Free tier caveats:** the free web service **cold-starts** after inactivity (first
  request is slow), and the free Postgres **expires after 90 days**. For anything
  lasting, bump both to a paid plan in `render.yaml` (`plan:`), then re-apply.
- **Rotate/revoke:** `KEYWAY_API_TOKEN` is a bearer token — treat it like a secret.
- Point discovery at your own configs by changing `--path` (or add `--in-cluster`
  with a kube-context) in `render.yaml`.
