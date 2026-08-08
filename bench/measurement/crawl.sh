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
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
corpus="$here/corpus"; mkdir -p "$corpus"
sources="$here/sources.tsv"
: "${GH_TOKEN:?set GH_TOKEN in the environment (do not hardcode a token)}"

MAX_PAGES="${MAX_PAGES:-1}"
MAX_PER_REPO="${MAX_PER_REPO:-5}"
declare -A seen repo_count

# Each query targets a distinct, real auth-config shape. All have a text term
# (code search requires one) plus a YAML qualifier.
queries=(
  '"kind: RequestAuthentication" language:YAML'   # Istio JWT validation
  '"jwt_authn" extension:yaml'                      # Envoy JWT filter
  '"request.auth.claims" language:YAML'            # Istio claim-based authz
  '"remote_jwks" extension:yaml'                    # Envoy remote JWKS provider
  '"kind: AuthorizationPolicy" "when" language:YAML' # Istio authz conditions
)

b64d() { python3 -c 'import sys,base64; sys.stdout.buffer.write(base64.b64decode(sys.stdin.read()))'; }

printf 'n\trepo\tpath\tsha\tlicense\turl\n' > "$sources"
n=0
for q in "${queries[@]}"; do
  for ((page=1; page<=MAX_PAGES; page++)); do
    resp="$(gh api -X GET search/code -f q="$q" -F per_page=100 -F page="$page" 2>/dev/null)" || break
    items="$(jq '.items | length' <<<"$resp" 2>/dev/null || echo 0)"
    [ "$items" -eq 0 ] && break
    while IFS=$'\t' read -r repo path sha priv lic url; do
      [ "$priv" = "true" ] && continue                       # public only (ethics)
      key="$repo/$path"; [ -n "${seen[$key]:-}" ] && continue; seen[$key]=1
      rc="${repo_count[$repo]:-0}"; [ "$rc" -ge "$MAX_PER_REPO" ] && continue
      repo_count[$repo]=$((rc+1))
      [ -z "$lic" ] || [ "$lic" = "null" ] && lic="NOASSERTION"
      safe="$(printf '%s__%s' "$repo" "$path" | tr '/ :' '___').yaml"
      if gh api "repos/$repo/git/blobs/$sha" --jq '.content' 2>/dev/null | b64d > "$corpus/$safe" 2>/dev/null && [ -s "$corpus/$safe" ]; then
        n=$((n+1))
        printf '%d\t%s\t%s\t%s\t%s\t%s\n' "$n" "$repo" "$path" "$sha" "$lic" "$url" >> "$sources"
      else
        rm -f "$corpus/$safe"
      fi
    done < <(jq -r '.items[] | [.repository.full_name, .path, .sha, (.repository.private|tostring), (.repository.license.spdx_id // "NOASSERTION"), .html_url] | @tsv' <<<"$resp")
    sleep 7   # code search REST is limited to ~10 req/min
  done
done
repos="$(tail -n +2 "$sources" | cut -f2 | sort -u | wc -l | tr -d ' ')"
echo "fetched $n files from $repos public repos into $corpus"
echo "manifest → $sources"
