# Feature Specification: Multiplatform Release Distribution

**Feature Branch**: `[005-multiplatform-release]`

**Created**: 2026-07-03

**Status**: Draft

**Input**: User description: "Contexto Atualmente o CLI só pode ser executado clonando o repositório e compilando localmente com go build, o que cria fricção para novos usuários. Queremos oferecer binários pré-compilados e métodos de instalação idiomáticos para cada sistema operacional, reduzindo o tempo entre 'descobri o projeto' e 'rodei o primeiro comando'. Objetivo Permitir que qualquer usuário instale o CLI em macOS, Linux e Windows sem precisar de Go instalado, usando o método de distribuição mais comum da sua plataforma."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - One-Command Install on macOS and Linux (Priority: P1)

A user on macOS or Linux discovers the project and wants to start using it immediately without installing a compiler or toolchain. They run a single install command in their terminal, the correct binary for their operating system and architecture is detected automatically, verified for integrity, and placed on their path.

**Why this priority**: This removes the biggest friction point for new users and directly addresses the goal of reducing time from discovery to first command.

**Independent Test**: A fresh macOS or Linux environment without Go installed can run the install command and invoke the CLI successfully.

**Acceptance Scenarios**:

1. **Given** a user on macOS (Intel or Apple Silicon) or Linux (amd64 or arm64), **When** they run the documented install command, **Then** the matching binary is downloaded, verified, and installed to a directory on their PATH.
2. **Given** the install script cannot write to the system directory, **When** it requests elevated permissions or falls back to a user-writable directory, **Then** the installation still completes successfully and the binary is executable.
3. **Given** a downloaded archive, **When** its checksum does not match the published checksum, **Then** the installer aborts with a clear error message and does not install the binary.

---

### User Story 2 - Manual Download from Release Page (Priority: P1)

A user prefers to download artifacts manually or is on a platform not covered by the install script. They visit the project's release page, choose the archive matching their operating system and architecture, download it, extract it, and run the binary.

**Why this priority**: Manual downloads cover Windows users and power users who want full control over where the binary is placed.

**Independent Test**: A user can locate the correct archive for their platform, download it, extract it, and run the CLI without additional runtime dependencies.

**Acceptance Scenarios**:

1. **Given** a release page with platform-specific archives, **When** a user selects the archive for their OS/architecture, **Then** the download contains a single executable binary plus standard documentation files.
2. **Given** a Windows user downloads the Windows archive, **When** they extract it and run the executable, **Then** the program starts without requiring external libraries or runtimes.
3. **Given** a published archive, **When** a user compares its SHA256 checksum against the published checksums file, **Then** the values match.

---

### User Story 3 - Version and Build Metadata for Support (Priority: P2)

A user or support person needs to identify exactly which build is running to reproduce an issue or confirm an update. They run the built-in version command and see the release version, source commit reference, and build date.

**Why this priority**: Reliable support and debugging depend on knowing the exact binary being executed, especially when distributing multiple builds across platforms.

**Independent Test**: Running the version command on any published binary displays the same three metadata values that correspond to the release it came from.

**Acceptance Scenarios**:

1. **Given** an installed binary from an official release, **When** the user runs the version command, **Then** the output includes the release version, commit reference, and build date.
2. **Given** a binary built locally without the release process, **When** the user runs the version command, **Then** the output indicates it is a development build rather than a stable release.

---

### User Story 4 - Package Manager Install on macOS via Homebrew (Priority: P2)

A macOS user who manages tools with Homebrew wants to install and update the CLI using the familiar `brew install` workflow. They add the project's tap and install the package, then upgrade it later with `brew upgrade`.

**Why this priority**: Homebrew is the dominant package manager on macOS and matches user expectations for idiomatic installation.

**Independent Test**: A macOS user with Homebrew can install the CLI via tap and run the version command without manually downloading archives.

**Acceptance Scenarios**:

1. **Given** a user has Homebrew installed, **When** they add the tap and install the package, **Then** the binary is available on PATH and matches the latest release.
2. **Given** a new release is published, **When** the tap formula is updated, **Then** `brew upgrade` installs the new version.

---

### User Story 5 - Fallback Install for Go Users (Priority: P3)

A user who already has the Go toolchain installed prefers the universal fallback. They run a single `go install` command using the module path and receive the latest version.

**Why this priority**: It provides a simple escape hatch for developers and early adopters regardless of platform-specific packages.

**Independent Test**: A system with Go installed can run the install command and obtain a working binary.

**Acceptance Scenarios**:

1. **Given** a user with Go installed, **When** they run the documented `go install` command, **Then** a working binary is placed in their Go bin directory.

---

### Edge Cases

- What happens when a user runs the install script on an unsupported operating system (e.g., Windows or FreeBSD)? The script should exit with a clear message pointing to the manual download page.
- What happens when the latest release has no asset for the detected architecture? The installer should report the unsupported combination and abort cleanly.
- What happens when network access is unavailable during install? The script should surface the network error without creating a partial installation.
- What happens when a user already has an older version installed? The install should overwrite the existing binary and update the version metadata.
- What happens when the install directory is not on PATH? The installer should warn the user to add it before the binary can be invoked by name.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST produce static binaries for the following platforms: macOS Intel (amd64), macOS Apple Silicon (arm64), Linux Intel (amd64), Linux ARM (arm64), and Windows Intel (amd64).
- **FR-002**: Each released binary MUST embed the release version, source commit reference, and build date so they can be inspected at runtime.
- **FR-003**: Creating a release tag in the format `vX.Y.Z` MUST automatically trigger a release pipeline that publishes platform-specific archives to the project's release page.
- **FR-004**: Every published archive MUST have an accompanying SHA256 checksum, and all checksums for a release MUST be published in a single checksums file.
- **FR-005**: The project MUST provide a shell install script for macOS and Linux that auto-detects the operating system and architecture, downloads the correct archive, verifies its checksum, and installs the binary.
- **FR-006**: The project MUST provide a Homebrew tap so macOS users can install the CLI with `brew install`.
- **FR-007**: The README MUST include an "Installation" section with clear instructions for macOS, Linux, and Windows, including the one-command install, manual download, and universal fallback methods.
- **FR-008**: The CLI MUST expose a `version` subcommand that prints the embedded release version, commit reference, and build date.
- **FR-009**: The install script MUST gracefully handle permission errors by either using elevated privileges or falling back to a user-writable directory, and it MUST inform the user of the chosen location.
- **FR-010**: The Windows archive MUST contain an `.exe` file that runs without external dependencies beyond the Windows operating system.

### Key Entities *(include if feature involves data)*

- **Release Artifact**: A platform-specific archive containing the CLI binary, README, and license. Attributes: target OS, target architecture, file format, SHA256 checksum, release version.
- **Release**: A versioned publication associated with a source tag. Attributes: semantic version, source commit reference, build date, collection of release artifacts, checksums file, changelog.
- **Install Script**: A shell program that resolves the latest release, selects the correct artifact, verifies integrity, and installs the binary. Attributes: supported platforms, default install directory, fallback behavior, checksum verification logic.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can install the CLI on macOS or Linux in under 60 seconds from discovering the install command.
- **SC-002**: Every stable release tag produces installable artifacts for all five supported OS/architecture combinations without manual intervention.
- **SC-003**: 100% of published release archives have a matching SHA256 checksum in the published checksums file.
- **SC-004**: The `version` command returns the same version string shown on the release page for every official release binary.
- **SC-005**: The Windows executable starts on a clean Windows system without requiring additional runtimes or libraries.
- **SC-006**: The README installation instructions cover at least three distinct methods: one-command shell install, manual archive download, and universal fallback.

## Assumptions

- The project uses semantic versioning tags prefixed with `v` (e.g., `v1.2.3`) to trigger releases.
- The project is hosted on GitHub, which provides release pages and artifact hosting.
- A project maintainer can create and manage a separate public repository for the Homebrew tap.
- Users installing via the shell script have `curl` and standard Unix utilities (`tar`, `shasum` or `sha256sum`) available.
- Users installing via Homebrew already have Homebrew configured on their macOS system.
- Code signing and notarization for macOS are out of scope for this feature and will be addressed separately.
- Auto-update functionality inside the CLI is out of scope for this feature.
