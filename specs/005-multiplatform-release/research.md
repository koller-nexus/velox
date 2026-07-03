# Research: Multiplatform Release Distribution

## Unknowns Resolved

### 1. Release Automation Tool

**Decision**: Use GoReleaser v2 configured via `.goreleaser.yaml`.

**Rationale**:
- GoReleaser is the de-facto standard for Go release automation on GitHub.
- It handles cross-compilation, archives, checksums, changelog, and Homebrew taps in a single config.
- GitHub Actions has an official `goreleaser/goreleaser-action@v6` that supports v2.
- Avoids maintaining brittle shell scripts for every release step.

**Alternatives considered**:
- Pure `make cross` + manual GitHub upload: more control but error-prone and does not generate checksums/tap automatically.
- Custom GitHub Actions matrix with `go build`: possible, but requires extra steps for archives, checksum aggregation, and release creation.

### 2. Ldflags Target

**Decision**: Continue injecting build metadata into `github.com/koller-nexus/velox/internal/version`.

**Rationale**:
- The package already declares `Version`, `Commit`, and `Date` variables.
- `cmd/velox/main.go` has no build variables; changing the entrypoint would require duplicate code and conflict with the existing `internal/version.String()` helper used by both `--version` and `version`.
- The user's sample config used `main.version` as an illustration, but the project already has a dedicated package.

**Ldflags**:
```text
-s -w
-X github.com/koller-nexus/velox/internal/version.Version={{.Version}}
-X github.com/koller-nexus/velox/internal/version.Commit={{.ShortCommit}}
-X github.com/koller-nexus/velox/internal/version.Date={{.Date}}
```

### 3. Supported Platforms

**Decision**: Build darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64.

**Rationale**:
- Matches the user's explicit scope (5 combinations).
- Windows/arm64 is excluded because it is listed as out of scope in the feature description and avoids untested Windows-on-ARM behavior.
- The existing `scripts/build.sh` already builds `windows/arm64`; this will be aligned with the release config.

### 4. Archive Formats

**Decision**: `.tar.gz` for macOS/Linux, `.zip` for Windows.

**Rationale**:
- Native archive formats for each platform (`tar` is available by default on Unix; Windows Explorer handles `.zip`).
- GoReleaser supports per-OS format overrides.

### 5. Install Script

**Decision**: Create `scripts/install.sh` as a POSIX-compatible shell script served from the repository root.

**Rationale**:
- POSIX `sh` maximizes compatibility across macOS (zsh default but `/bin/sh` exists), Linux distros, and minimal containers.
- Must detect `uname -s` and `uname -m`, resolve latest release via GitHub API, verify SHA256, and install to `/usr/local/bin` or `~/.local/bin` fallback.
- `sudo` fallback is acceptable but must be explicit and not silent.

### 6. Homebrew Tap

**Decision**: Configure GoReleaser `brews` publisher pointing to a separate `koller-nexus/homebrew-tap` repository.

**Rationale**:
- Requires `HOMEBREW_TAP_TOKEN` secret; the tap must exist before the first release.
- If the token is absent, GoReleaser will fail the release unless the block is commented out; the user opted to include it in this first version.
- Formula will install the `velox` binary and test with `velox version`.

### 7. Go Version in CI

**Decision**: Pin `go-version: "1.26.4"` in `release.yml` to match `go.mod`.

**Rationale**:
- Ensures reproducible builds and avoids surprises when a new Go minor is released.
- Matches the existing `Makefile` and `go.mod`.

### 8. README Update Strategy

**Decision**: Add a dedicated "Installation" section before "Usage" with three methods: shell installer, Homebrew, manual download, and `go install` fallback.

**Rationale**:
- Keeps existing users' reference material intact.
- Windows instructions will reference the `.zip` download from the release page.

## Decisions Summary

| Topic | Decision | Notes |
|-------|----------|-------|
| Release tool | GoReleaser v2 | `.goreleaser.yaml` + GitHub Actions |
| Build metadata package | `internal/version` | Existing variables, no change to `main.go` |
| Platforms | 5 targets | darwin/amd64, darwin/arm64, linux/amd64, linux/arm64, windows/amd64 |
| Archives | tar.gz (Unix), zip (Windows) | Format override in GoReleaser |
| Checksum | SHA256 in `checksums.txt` | GoReleaser `checksum` block |
| Installer | `scripts/install.sh` | POSIX sh, curl-based, checksum verification |
| Homebrew | Separate tap repo | `koller-nexus/homebrew-tap` |
| CI Go version | 1.26.4 | Match `go.mod` |
