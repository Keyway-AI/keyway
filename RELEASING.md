# Releasing Keyway

A release is cut by **pushing a semver tag**. Everything else is automated by
[`.github/workflows/release.yml`](.github/workflows/release.yml), which on a
`v*` tag:

- builds cross-platform binaries (`linux`/`darwin` × `amd64`/`arm64`),
- builds and pushes the container image to
  `ghcr.io/nometria/keyway:vX.Y.Z` and `:latest`,
- publishes a **GitHub Release** with the binary tarballs + `checksums.txt`
  attached and auto-generated notes.

## Checklist

1. **Green main.** Be on an up-to-date `main` with CI passing.

2. **Update the changelog.** Move the items under `## [Unreleased]` in
   [`CHANGELOG.md`](CHANGELOG.md) into a new `## [X.Y.Z] - YYYY-MM-DD` section,
   refresh the compare/tag links at the bottom, and commit:

   ```bash
   git commit -am "docs(changelog): release vX.Y.Z"
   git push
   ```

3. **Tag and push.** The tag name is the version and drives every artifact:

   ```bash
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. **Watch the Release workflow.** On success it publishes the image, the
   binaries, and the GitHub Release. Pre-release suffixes (`-rc.1`, `-beta.1`)
   are marked as GitHub pre-releases automatically.

5. **Smoke-test the published image:**

   ```bash
   docker run -p 8080:8080 ghcr.io/nometria/keyway:vX.Y.Z
   # open http://localhost:8080 — the UI should load on sample data
   ```

6. **First public release only:** make the repository public, then confirm the
   README badges (CI, Go Reference, Go Report Card) resolve — they only render
   once the repo is public and has a run / pkg.go.dev entry.

## Versioning

- Pre-1.0, breaking changes may land in **minor** bumps — call them out in the
  changelog's `### Changed` section.
- Use `-rc.N` / `-beta.N` suffixes for pre-releases.
- The binary reports its version via `keyway version` (injected at build time
  from the tag through `-ldflags`).
