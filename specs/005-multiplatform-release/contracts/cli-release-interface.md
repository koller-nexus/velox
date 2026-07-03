# CLI Release Interface Contract

## Command: `velox version`

**Purpose**: Print build metadata for support and verification.

**Output format** (stdout, plain text):
```text
velox <version> (commit <short-commit>, built <date>, go<go-version> <os>/<arch>)
```

**Example**:
```text
velox 1.2.3 (commit a1b2c3d, built 2026-07-03T11:00:00Z, go1.26.4 darwin/arm64)
```

**Requirements**:
- Works offline.
- Exit code `0`.
- Same output as `velox --version`.
- Injected at build time via `-ldflags` into `github.com/koller-nexus/velox/internal/version`.

## Command: `velox --version`

**Purpose**: Flag alias for `velox version`.

**Output format**: Identical to `velox version`.

**Requirements**:
- Works offline.
- Exit code `0`.

## Install Script Contract

**Invocation**:
```bash
curl -fsSL https://raw.githubusercontent.com/koller-nexus/velox/main/scripts/install.sh | sh
```

**Environment variables**:
| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | `latest` | Version tag to install (e.g. `v1.2.3`) |
| `INSTALL_DIR` | `/usr/local/bin` | Target directory for the binary |

**Behavior**:
1. Detect OS (`darwin`, `linux`) and architecture (`amd64`, `arm64`).
2. Resolve version: query GitHub API for latest release if `VERSION=latest`.
3. Download the matching archive and `checksums.txt` to a temp directory.
4. Verify the archive SHA256 against `checksums.txt`.
5. Extract the binary.
6. Move the binary to `INSTALL_DIR`, using `sudo` if the directory is not writable.
7. Run `velox version` to confirm installation.

**Error behavior**:
- Unsupported OS/arch: print error and exit non-zero.
- Network failure: surface curl error and exit non-zero.
- Checksum mismatch: print error, do not install, exit non-zero.

## Release Artifact Contract

**Naming**:
```text
velox_<version>_<os>_<arch>.<ext>
```

**Examples**:
- `velox_1.2.3_darwin_amd64.tar.gz`
- `velox_1.2.3_linux_arm64.tar.gz`
- `velox_1.2.3_windows_amd64.zip`

**Contents**:
- `velox` or `velox.exe`
- `README.md`
- `LICENSE`

**Checksums file**:
- Name: `checksums.txt`
- Format: `<sha256>  <filename>`

## Homebrew Formula Contract

**Tap repository**: `koller-nexus/homebrew-tap`

**Install command**:
```bash
brew tap koller-nexus/tap
brew install velox
```

**Formula behavior**:
- Downloads the matching macOS archive from the GitHub release.
- Installs the `velox` binary to the Homebrew prefix.
- Runs `velox version` as a smoke test.

## GitHub Actions Trigger Contract

**Trigger**: Push a tag matching `v*`.

**Permissions**:
- `contents: write`

**Secrets**:
| Secret | Required | Purpose |
|--------|----------|---------|
| `GITHUB_TOKEN` | Yes | Create release and upload artifacts |
| `HOMEBREW_TAP_TOKEN` | Yes for Homebrew | Push updated formula to tap repo |

**Artifacts produced**:
- Five platform archives.
- `checksums.txt`.
- Updated Homebrew formula (if tap configured).
