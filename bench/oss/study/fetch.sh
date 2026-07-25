#!/usr/bin/env bash
# Re-download the 60-repo independent benchmark manifests from their pinned
# commits (needs an authenticated `gh`). Then: keyway discover + grade.py.
set -uo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
out="$here/manifests"; mkdir -p "$out"
tail -n +2 "$here/sources.tsv" | while IFS=$'\t' read -r n repo path url; do
  [ -z "$url" ] && continue
  gh api "$url" --jq '.content' 2>/dev/null | base64 --decode > "$out/$(printf '%s' "$n")-$(echo "$repo" | tr '/' '__').yaml"
done
echo "fetched $(ls "$out" | wc -l) manifests into $out"
echo "now run:  go run ./cmd/keyway discover --path $out --output json > /tmp/kw.json && python3 $here/grade.py $out /tmp/kw.json"
