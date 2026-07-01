# Feature Specification: Help Commands & Additional Useful Commands

**Feature Branch**: `003-cli-help-commands`

**Created**: 2026-07-01

**Status**: Draft

**Input**: User description: "Criar comandos de help, e adicionar comandos a mais, uteis para o uso"

## Clarifications

### Session 2026-07-01

- Q: Which additional commands are in scope for this feature (beyond the P1 help system)? → A: All four candidates — a `version` subcommand, a nearby-servers listing, a local-state/config inspection command, and a quick latency-only check.
- Q: How should the quick latency-only command measure latency and jitter (FR-013)? → A: Reuse the ndt7 connection and sample round-trip time over a short window, skipping the bulk download/upload phases (no ICMP, no elevated privileges).
- Q: Which new commands must support `--json` machine-readable output (FR-014)? → A: The nearby-servers listing, the local-state/config command, and the latency-only command support `--json`; the `version` subcommand stays plain text (`--version` already covers it).
- Q: How many servers should the nearby-servers listing show by default (FR-011)? → A: The server velox would select plus the next few nearest alternatives — about 5 in total.
- Q: What sampling window and overall time budget should the latency-only command use (FR-013)? → A: Sample round-trip time for about 5 seconds; the overall command budget defaults to 10 seconds and is overridable via `--timeout`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Discoverable, command-aware help (Priority: P1)

A person who just installed velox runs it without knowing the available
commands. They type `velox help` and see a clear overview of every command with
a one-line description. They then type `velox help consent` (or `velox consent
--help`) and get the detailed usage — synopsis, options, and a short
explanation — for that specific command, so they can use it correctly without
reading external documentation.

**Why this priority**: This is the explicit primary request ("criar comandos de
help"). Discoverability is the foundation for every other command in the tool;
without it, added commands are effectively hidden. It delivers standalone value
even if no new functional commands are added.

**Independent Test**: Run `velox help`, `velox help <command>`, and `<command>
--help` on a machine with no network access and confirm each prints complete,
correct usage to stdout and exits successfully — no measurement or location
lookup occurs.

**Acceptance Scenarios**:

1. **Given** a fresh install, **When** the user runs `velox help`, **Then** the
   tool prints an overview listing every available command with a one-line
   description and exits with code 0, touching no network.
2. **Given** any known command name, **When** the user runs `velox help
   <command>`, **Then** the tool prints that command's synopsis, options, and
   description to stdout and exits 0.
3. **Given** any known command, **When** the user runs `<command> --help`,
   **Then** the tool prints the same detailed usage as `velox help <command>`
   and exits 0 without performing the command's normal action.
4. **Given** an unknown command or an unknown flag, **When** the user runs it,
   **Then** the tool writes a clear error to stderr, suggests running `velox
   help`, and exits with the usage-error code.

---

### User Story 2 - More useful commands for everyday use (Priority: P2)

Beyond running a full speed test, users want lightweight, focused commands for
common needs: checking the tool version, seeing which nearby servers velox would
pick, finding where velox stores its local state, and doing a quick
latency-only check without a full download/upload run. Each command is a small,
self-contained action reachable from the command line.

**Why this priority**: This is the secondary request ("adicionar comandos a
mais, uteis para o uso"). It builds on the help system (P1) so the new commands
are discoverable, and it increases day-to-day utility without changing the core
measurement flow.

**Independent Test**: With the help system in place, invoke each new command and
confirm it performs its focused action, produces output on stdout, respects the
existing consent gate for any location lookup, and returns a meaningful exit
code — independently of the full `--check-internet` flow.

**Acceptance Scenarios**:

1. **Given** any environment, **When** the user runs the version command as a
   subcommand, **Then** the tool prints the same version line as `--version` and
   exits 0 without network access.
2. **Given** consent handling unchanged, **When** the user asks to list nearby
   servers, **Then** the tool shows the server it would select plus the next few
   nearest alternatives (about 5 total) with name, location, and approximate
   distance — without running a measurement, honoring the consent gate before
   any location lookup.
3. **Given** any environment, **When** the user asks where velox keeps its data,
   **Then** the tool reports the local paths for its stored consent decision and
   configuration and the effective settings, without network access.
4. **Given** a user who only cares about latency, **When** they run a
   quick-latency command, **Then** the tool reports latency and jitter without
   performing the download and upload phases.

> **Scope note**: All four candidate commands below are confirmed in scope
> (see Clarifications, Session 2026-07-01): version subcommand, nearby-servers
> listing, local-state/config inspection, and quick latency-only check.

---

### Edge Cases

- Running `velox help <unknown>` reports a usage error and points the user to
  `velox help`; it does not crash or run a measurement.
- `velox help help` returns help for the help command itself.
- `--help` combined with other flags takes precedence: usage is printed and no
  measurement, consent prompt, or state change occurs.
- Help and version work fully offline (no network dependency), matching the
  project's guarantee that `--help`/`--version` always work without network.
- A command that needs location (e.g., listing nearby servers) run
  non-interactively or with consent denied degrades gracefully with a clear
  message rather than blocking or performing a lookup.
- Every command that exists is reachable and documented in the overview; a
  command must not exist without appearing in `velox help`.

## Requirements *(mandatory)*

### Functional Requirements

**Help system (P1)**

- **FR-001**: The tool MUST provide a `help` command that prints an overview of
  every available command, each with a one-line description, to stdout, and
  exits with success.
- **FR-002**: The tool MUST provide `help <command>` that prints the named
  command's detailed usage (synopsis, options, and description) to stdout.
- **FR-003**: Every command MUST accept a `--help` flag that prints that
  command's detailed usage and exits with success, without performing the
  command's normal action, any network call, any consent prompt, or any state
  change.
- **FR-004**: When given an unknown command or unknown flag, the tool MUST write
  a clear, specific error to stderr, suggest running `velox help`, and exit with
  the usage-error code (distinct from success and from measurement failure).
- **FR-005**: The tool MUST preserve existing behavior: bare invocation and
  `--help` print the top-level overview, `--version` prints the version, and the
  existing `consent` subcommand keeps working (backward compatible).
- **FR-006**: All help and usage text MUST go to stdout; all error and
  diagnostic messages MUST go to stderr.
- **FR-007**: The command overview MUST stay complete — every command the tool
  accepts (existing and new) MUST appear in `velox help`, so no command is
  hidden from discovery.
- **FR-008**: Help, version, and config output MUST NOT require network access
  and MUST NOT trigger a location lookup or consent prompt.

**Additional commands (P2)**

- **FR-009**: The additional commands in scope for this feature are the four
  described in FR-010 through FR-013 (confirmed in Clarifications, Session
  2026-07-01).
- **FR-010**: The tool MUST provide a `version` subcommand that prints the same
  output as the `--version` flag, for users who type the command form.
- **FR-011**: The tool MUST provide a command that lists the nearby candidate
  test servers it would select — the server it would pick plus the next few
  nearest alternatives, about 5 in total — showing each server's name,
  location, and approximate distance, without running a measurement, and
  honoring the existing consent gate before any location lookup.
- **FR-012**: The tool MUST provide a command that reports where velox stores
  its local state (the consent decision and configuration) and the effective
  configuration values, without network access.
- **FR-013**: The tool MUST provide a quick latency-only command that reports
  latency and jitter without performing the download and upload phases. It MUST
  obtain these values by opening the same ndt7 connection used by the full test
  and sampling round-trip time over a short window (no ICMP probing, no elevated
  privileges), preserving the tool's existing latency (minimum RTT) and jitter
  definitions. The sampling window MUST be short (about 5 seconds) so the
  command returns promptly, and the overall command budget MUST default to 10
  seconds, overridable via `--timeout`.
- **FR-014**: Every new command MUST follow the tool's CLI contract: results to
  stdout, diagnostics/errors to stderr, and meaningful exit codes (success,
  usage error, measurement/network failure). The nearby-servers listing
  (FR-011), the local-state/config command (FR-012), and the latency-only
  command (FR-013) MUST support a `--json` machine-readable output mode
  (consistent with the existing `--check-internet --json` behavior); the
  `version` subcommand (FR-010) MAY remain plain text since `--version` already
  covers the machine-parseable single-line form.

### Key Entities *(include if data involved)*

- **Command**: A named, invocable capability of the tool. Attributes: name,
  short (one-line) description, detailed synopsis, options/flags, and usage
  examples. The help system reads from the full set of commands so the overview
  and per-command help stay consistent with what the tool actually accepts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can discover 100% of available commands and their
  purpose from `velox help` alone, without consulting external documentation.
- **SC-002**: For every command, the usage shown by `velox help <command>` and
  by `<command> --help` is complete and identical.
- **SC-003**: Help and version commands return their output in under 1 second
  and succeed with no network connectivity available.
- **SC-004**: Every mistyped command or flag produces an actionable error that
  names the problem and points to `velox help`, and returns a non-zero exit
  code, 100% of the time.
- **SC-005**: Users can inspect version, intended server selection, local state
  locations, and a latency-only reading without running a full download/upload
  test.
- **SC-006**: On a working connection, the latency-only command returns a result
  in under 10 seconds.

## Assumptions

- Existing behavior is preserved: `velox --check-internet`, `velox --version`,
  `velox --help`, bare invocation, and `velox consent` all keep working exactly
  as they do today; this feature adds to the command surface rather than
  replacing it.
- Help, version, and local-state inspection commands perform no network I/O and
  never trigger a location lookup or consent prompt, consistent with the
  project's guarantee that `--help`/`--version` work offline.
- Additional commands reuse the tool's existing measurement, server-location,
  and consent components; no new external runtime dependencies are introduced
  (consistent with the project's minimal-dependency principle).
- Any command that lists nearby servers or otherwise needs location goes through
  the existing consent gate and degrades gracefully (clear message, fallback)
  when consent is denied or the environment is non-interactive.
- The command form of version (`velox version`) is additive and does not remove
  the `--version` flag.
- "Useful commands" is scoped to lightweight, self-contained actions that fit
  the existing tool; larger features (e.g., persistent run history, scheduled
  monitoring, remote reporting) are out of scope for this feature.
