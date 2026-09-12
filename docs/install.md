# Installing Keyway

Keyway is a single static binary (`keyway`), plus a container image that also serves
the web UI. Pick whichever fits.

## 1. One-line installer (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/Keyway-AI/keyway/main/install.sh | sh
```

It detects your OS/arch, downloads the matching release tarball, **verifies its
SHA-256 checksum**, and installs `keyway` (and `keyway-runner`) into `/usr/local/bin`
— or `$HOME/.local/bin` if that is not writable.

Options:

```bash
# pin a version
KEYWAY_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/Keyway-AI/keyway/main/install.sh | sh

# choose the install directory
KEYWAY_BINDIR="$HOME/bin" curl -fsSL https://raw.githubusercontent.com/Keyway-AI/keyway/main/install.sh | sh
```

Prefer to read before you pipe to a shell? Download `install.sh`, review it, then run it.

## 2. Go

```bash
go install github.com/Keyway-AI/keyway/cmd/keyway@latest
```

Installs into `$(go env GOPATH)/bin`. Requires Go 1.25+.

## 3. Container image

The image runs the API and the embedded web UI:

```bash
docker run -p 8080:8080 ghcr.io/keyway-ai/keyway        # sample-data demo on :8080
```

Images are multi-arch, published on every release, and **cosign-signed** (keyless):

```bash
cosign verify ghcr.io/keyway-ai/keyway:latest \
  --certificate-identity-regexp 'https://github.com/Keyway-AI/keyway' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## 4. Prebuilt binaries (manual)

Download from the [releases page](https://github.com/Keyway-AI/keyway/releases). Each
release ships `keyway_<version>_<os>_<arch>.tar.gz`, a `checksums.txt`, and an SBOM.

```bash
V=v0.1.0; OS=darwin; ARCH=arm64
curl -fsSLO "https://github.com/Keyway-AI/keyway/releases/download/$V/keyway_${V}_${OS}_${ARCH}.tar.gz"
curl -fsSLO "https://github.com/Keyway-AI/keyway/releases/download/$V/checksums.txt"
shasum -a 256 -c checksums.txt --ignore-missing        # verify
tar -xzf "keyway_${V}_${OS}_${ARCH}.tar.gz"
sudo install -m 0755 keyway /usr/local/bin/keyway
```

## 5. From source

```bash
git clone https://github.com/Keyway-AI/keyway && cd keyway
make build            # -> ./bin/keyway
```

## Verify it works

```bash
keyway version
keyway --help
echo "$AGENT_TOKEN" | keyway agent inspect --audience https://mcp.example/api
```

## Uninstall

Remove the binaries from wherever you installed them, e.g.:

```bash
rm -f /usr/local/bin/keyway /usr/local/bin/keyway-runner
```

## Homebrew

Not yet — a tap is on the roadmap. For now use the one-line installer or `go install`.
Track it in [Discussions](https://github.com/Keyway-AI/keyway/discussions).
