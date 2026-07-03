# Data Model: Multiplatform Release Distribution

This feature does not introduce new application data or persistent state. The entities below describe the release and installation domain.

## Entities

### Release

Represents a single versioned publication of velox.

| Field | Type | Description |
|-------|------|-------------|
| version | string | Semantic version with `v` prefix, e.g. `v1.2.3` |
| commit | string | Short git commit hash the release was built from |
| date | string | Build timestamp in RFC3339/ISO8601 UTC format |
| artifacts | []ReleaseArtifact | Platform-specific archives |
| checksumsFile | string | URL/path to `checksums.txt` |
| changelog | string | Generated changelog from commit history |

### ReleaseArtifact

A downloadable archive for one OS/architecture combination.

| Field | Type | Description |
|-------|------|-------------|
| os | string | Target operating system: `darwin`, `linux`, `windows` |
| arch | string | Target architecture: `amd64`, `arm64` |
| format | string | Archive format: `tar.gz`, `zip` |
| filename | string | Archive name, e.g. `velox_1.2.3_linux_amd64.tar.gz` |
| binaryName | string | Executable name inside the archive: `velox` or `velox.exe` |
| sha256 | string | Hex SHA256 digest of the archive |
| url | string | Download URL on GitHub Releases |

### InstallScriptConfig

Parameters that control the shell installer.

| Field | Type | Description |
|-------|------|-------------|
| repoOwner | string | GitHub organization/user: `koller-nexus` |
| repoName | string | Repository name: `velox` |
| binaryName | string | Installed binary name: `velox` |
| defaultDir | string | Preferred install directory: `/usr/local/bin` |
| fallbackDir | string | User-writable fallback: `~/.local/bin` |
| supportedOs | []string | `darwin`, `linux` |
| supportedArch | []string | `amd64`, `arm64` |

## Validation Rules

- A release tag must match `v*`, e.g. `v1.2.3`.
- Every `ReleaseArtifact` must have a matching entry in `checksums.txt`.
- The install script must abort if `OS` or `ARCH` is unsupported.
- The install script must abort if the downloaded archive checksum does not match.

## State Transitions

N/A — release creation is an idempotent pipeline triggered by a git tag.
