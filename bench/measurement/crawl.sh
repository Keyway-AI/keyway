#!/usr/bin/env bash
# Crawl public GitHub for real JWT/auth deployment config → build the Paper A
# measurement corpus. READ-ONLY, public repos only. Records repo + blob SHA +
# license for every artifact so the corpus is reproducible and attributable.
#
# Auth: reads GH_TOKEN from the environment — NEVER hardcode a token here or
# commit one. Run:  GH_TOKEN=... bash bench/measurement/crawl.sh
#
# Tunables (env): MAX_PAGES (pages/query, 100 results each), MAX_PER_REPO
# (cap files from any one repo, to avoid a monorepo dominating the sample).
#
# Resume-friendly: re-running APPENDS. It preloads the existing sources.tsv so
# already-fetched (repo, path) pairs are skipped and the per-repo cap carries
# across runs. Use this to finish an interrupted crawl, or to grow the corpus by
# re-running with a higher MAX_PAGES / MAX_PER_REPO. Delete sources.tsv (and the
# corpus) to start fresh.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
corpus="$here/corpus"; mkdir -p "$corpus"
sources="$here/sources.tsv"
: "${GH_TOKEN:?set GH_TOKEN in the environment (do not hardcode a token)}"

MAX_PAGES="${MAX_PAGES:-1}"
MAX_PER_REPO="${MAX_PER_REPO:-5}"
declare -A seen repo_count

# Code search returns at most ~1000 results per query. The two highest-volume
# shapes (Istio RequestAuthentication, Envoy jwt_authn) are therefore PARTITIONED
# by file size into non-overlapping buckets, each under the cap, so the union
# captures far more than one flat query can. The rest add distinct field- and
# claim-level shapes, all within what the discoverers parse (Istio RA /
# AuthorizationPolicy, Envoy jwt_authn). Every query has a text term (code search
# requires one) plus language:YAML. Add size buckets to any shape that saturates.
queries=(
  # -- Istio RequestAuthentication (total ~3072). Buckets calibrated to the real
  #    size distribution so each is a small, fully-retrievable window. The genuine
  #    per-service configs are <=10KB (~1329); files >10KB are mostly install
  #    bundles / must-gather dumps that --dedup + --exclude-examples strip. --
  '"kind: RequestAuthentication" language:YAML size:<600'
  '"kind: RequestAuthentication" language:YAML size:600..1800'
  '"kind: RequestAuthentication" language:YAML size:1800..3000'
  '"kind: RequestAuthentication" language:YAML size:3000..5000'
  '"kind: RequestAuthentication" language:YAML size:5000..10000'
  '"kind: RequestAuthentication" language:YAML size:>10000'
  '"jwtRules" language:YAML'                          # RA field; catches templated/edge RA
  # -- Istio AuthorizationPolicy: JWT claim conditions (request.auth.* family) --
  '"request.auth.claims" language:YAML'
  '"request.auth.audiences" language:YAML'
  '"request.auth.presenter" language:YAML'
  '"request.auth.principals" language:YAML'
  '"kind: AuthorizationPolicy" "when" language:YAML'
  # -- Envoy jwt_authn (total ~1208), size-partitioned + provider/field variants --
  '"jwt_authn" language:YAML size:<800'
  '"jwt_authn" language:YAML size:800..2500'
  '"jwt_authn" language:YAML size:2500..10000'
  '"jwt_authn" language:YAML size:>10000'
  '"remote_jwks" language:YAML'                       # Envoy remote JWKS provider
  '"local_jwks" language:YAML'                        # Envoy inline (local) JWKS provider
  '"JwtAuthentication" language:YAML'                 # Envoy jwt_authn typed_config @type
  '"payload_in_metadata" language:YAML'               # Envoy jwt_authn payload field
)

b64d() { python3 -c 'import sys,base64; sys.stdout.buffer.write(base64.b64decode(sys.stdin.read()))'; }

# Resume: if a manifest already exists, preload fetched (repo,path) pairs and the
# per-repo counts so this run skips them and appends; otherwise start a fresh file.
n=0
if [ -s "$sources" ] && [ "$(wc -l < "$sources")" -gt 1 ]; then
  while IFS=$'\t' read -r rn repo path _; do
    [ "$rn" = "n" ] && continue                      # header row
    [ -z "$repo" ] && continue
    seen["$repo/$path"]=1
    repo_count["$repo"]=$(( ${repo_count["$repo"]:-0} + 1 ))
    case "$rn" in ''|*[!0-9]*) ;; *) [ "$rn" -gt "$n" ] && n="$rn" ;; esac
  done < "$sources"
  echo "resuming: $n files already in $(basename "$sources") (${#seen[@]} unique paths), skipping those"
else
  printf 'n\trepo\tpath\tsha\tlicense\turl\n' > "$sources"
fi
new=0
for q in "${queries[@]}"; do
  for ((page=1; page<=MAX_PAGES; page++)); do
    # Fetch one search page, retrying with backoff on a transient failure or a
    # secondary rate limit. A bare failure previously broke the WHOLE query loop,
    # truncating each query to a few hundred of its ~1000 available results.
    resp=""; tries=0
    while ! { resp="$(gh api -X GET search/code -f q="$q" -F per_page=100 -F page="$page" 2>/dev/null)" && [ -n "$resp" ]; }; do
      tries=$((tries+1)); [ "$tries" -ge 4 ] && break
      echo "  search retry $tries (q='$q' page=$page): backing off $((tries*30))s"
      sleep $((tries*30))
    done
    [ -z "$resp" ] && break                 # still failing after retries; a re-run resumes it
    items="$(jq '.items | length' <<<"$resp" 2>/dev/null || echo 0)"
    [ "$items" -eq 0 ] && break             # past the last page of results for this query
    while IFS=$'\t' read -r repo path sha priv lic url; do
      [ "$priv" = "true" ] && continue                       # public only (ethics)
      key="$repo/$path"; [ -n "${seen[$key]:-}" ] && continue; seen[$key]=1
      rc="${repo_count[$repo]:-0}"; [ "$rc" -ge "$MAX_PER_REPO" ] && continue
      repo_count[$repo]=$((rc+1))
      [ -z "$lic" ] || [ "$lic" = "null" ] && lic="NOASSERTION"
      # Filesystem-safe, length-capped name: repo+path can be very long or carry
      # control chars (even CR), so map anything but [A-Za-z0-9._-] to '_' and, when
      # too long for the 255-byte limit, truncate and append a hash for uniqueness.
      safe="$(printf '%s__%s' "$repo" "$path" | tr -c 'A-Za-z0-9._-' '_')"
      if [ "${#safe}" -gt 180 ]; then
        safe="${safe:0:160}__$(printf '%s' "$repo/$path" | shasum | cut -c1-12)"
      fi
      safe="${safe}.yaml"
      bt=0; ok=0
      while [ "$bt" -lt 3 ]; do
        if gh api "repos/$repo/git/blobs/$sha" --jq '.content' 2>/dev/null | b64d > "$corpus/$safe" 2>/dev/null && [ -s "$corpus/$safe" ]; then ok=1; break; fi
        bt=$((bt+1)); sleep $((bt*5))      # retry blob on transient/secondary-limit failure
      done
      if [ "$ok" = 1 ]; then
        n=$((n+1)); new=$((new+1))
        printf '%d\t%s\t%s\t%s\t%s\t%s\n' "$n" "$repo" "$path" "$sha" "$lic" "$url" >> "$sources"
      else
        rm -f "$corpus/$safe"
      fi
    done < <(jq -r '.items[] | [.repository.full_name, .path, .sha, (.repository.private|tostring), (.repository.license.spdx_id // "NOASSERTION"), .html_url] | @tsv' <<<"$resp")
    sleep 7   # code search REST is limited to ~10 req/min
  done
done
repos="$(tail -n +2 "$sources" | cut -f2 | sort -u | wc -l | tr -d ' ')"
echo "manifest now holds $n files from $repos public repos (new this run: $new) in $corpus"
echo "manifest → $sources"
