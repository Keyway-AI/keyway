/**
 * Read uploaded YAML manifests into the `{path: content}` map the Cloud analyze
 * endpoint expects. We use the browser-relative path when available (so a picked
 * folder keeps its structure) and cap total size to keep requests sane — the
 * backend re-validates and only reads .yaml/.yml anyway.
 */
const MAX_FILES = 200;
const MAX_TOTAL_BYTES = 8 * 1024 * 1024; // 8 MiB across all files

export async function readManifestFiles(list: FileList): Promise<Record<string, string>> {
  const out: Record<string, string> = {};
  let total = 0;
  let count = 0;
  for (const f of Array.from(list)) {
    // webkitRelativePath is populated for directory picks; empty for plain files.
    const name = f.webkitRelativePath || f.name;
    if (!/\.ya?ml$/i.test(name)) continue;
    if (++count > MAX_FILES) throw new Error(`Too many files (max ${MAX_FILES}).`);
    total += f.size;
    if (total > MAX_TOTAL_BYTES) throw new Error("Upload is too large (max 8 MiB total).");
    out[name] = await f.text();
  }
  if (Object.keys(out).length === 0) {
    throw new Error("No YAML manifests found (.yaml / .yml).");
  }
  return out;
}
