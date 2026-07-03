# Quickstart: Validate Multiplatform Release Distribution

## Prerequisites

- Repository cloned locally.
- Go 1.26.4 installed.
- GoReleaser v2 installed (for local snapshot).
- GitHub repository access to create tags and secrets (for real release).
- Optional: `shellcheck` for installer validation.

## 1. Verify Current Build Metadata

Build the binary locally and confirm the version command prints metadata:

```bash
make build
./bin/velox version
./bin/velox --version
```

Expected output format:
```text
velox 0.1.0-dev (commit none, built 2026-07-03T11:00:00Z, go1.26.4 darwin/arm64)
```

## 2. Validate Cross-Compilation

Run the existing cross-compile script to ensure all targets build:

```bash
make cross
ls -la dist/
```

Expected artifacts:
```text
velox-darwin-amd64
velox-darwin-arm64
velox-linux-amd64
velox-linux-arm64
velox-windows-amd64.exe
```

## 3. Validate GoReleaser Snapshot

Run a local snapshot release without publishing:

```bash
goreleaser release --snapshot --clean
ls -la dist/
```

Expected artifacts include archives, checksums, and metadata:
```text
velox_0.1.0-dev_darwin_amd64.tar.gz
velox_0.1.0-dev_linux_amd64.tar.gz
...
checksums.txt
metadata.json
```

## 4. Validate Install Script

Run `shellcheck` on the installer:

```bash
shellcheck scripts/install.sh
```

Then run a local install from a snapshot release. This requires a fake GitHub release or local file serving; the simplest validation is to set `VERSION` to a known tag after a test release.

## 5. Validate End-to-End Release Flow

1. Push a test tag:
   ```bash
   git tag v0.0.0-test
   git push origin v0.0.0-test
   ```
2. Wait for the GitHub Actions `release.yml` workflow to complete.
3. Visit the release page and verify:
   - Five archives are present.
   - `checksums.txt` is present.
   - Homebrew formula was updated (if tap token is set).
4. Download one archive and verify its checksum:
   ```bash
   sha256sum velox_0.0.0-test_linux_amd64.tar.gz
   grep velox_0.0.0-test_linux_amd64.tar.gz checksums.txt
   ```
5. Extract and run:
   ```bash
   tar -xzf velox_0.0.0-test_linux_amd64.tar.gz
   ./velox version
   ```

## 6. Validate Homebrew Install (macOS)

After a real release, on a macOS machine with Homebrew:

```bash
brew tap koller-nexus/tap
brew install velox
velox version
```

## 7. Validate curl|sh Install (macOS/Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/koller-nexus/velox/main/scripts/install.sh | sh
velox version
```

## 8. Cleanup

Delete test tags and draft releases after validation:

```bash
git push --delete origin v0.0.0-test
git tag -d v0.0.0-test
```
