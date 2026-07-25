# 60-repo independent discovery study

Real Istio/Envoy JWT-auth manifests from **60 distinct public repos** (none in
our corpus), used to measure how Keyway's discovery generalises. See the full
write-up and results in [`docs/independent-benchmark.md`](../../../docs/independent-benchmark.md).

- `sources.tsv` — the 60 repos + file paths + commit-pinned download URLs (attribution).
- `fetch.sh` — re-downloads the manifests from those pinned commits (needs `gh`).
- `grade.py` — grades Keyway's discovery against an independent YAML parse
  (recall of every declared issuer / audience / claim).
- `results.json` — the scored result at the time of writing.

Reproduce: `bash fetch.sh` then follow its printed command (or `make bench-oss-study`).
Manifests are fetched, not vendored, to avoid bundling 60 projects' files.
