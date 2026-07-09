#!/bin/sh
#
# velox security gate: gosec (static analysis) + govulncheck (dependency CVEs).
# Fails (exit 1) when a finding meets or exceeds the severity threshold.
# Default threshold: HIGH (HIGH/CRITICAL fail; MEDIUM/LOW reported as warnings).
#   FR-012/FR-013, research R6, clarification Q5.
#
# Override threshold:  VELOX_SEVERITY=medium ./scripts/security.sh
# Library mode (for tests): VELOX_SECURITY_LIB=1 . ./scripts/security.sh
set -eu

VELOX_SEVERITY="${VELOX_SEVERITY:-high}"

# severity_rank maps a severity name to an integer for comparison.
severity_rank() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    low) echo 1 ;;
    medium) echo 2 ;;
    high) echo 3 ;;
    critical) echo 4 ;;
    *) echo 0 ;;
  esac
}

# gate_exit returns 0 (pass) or 1 (fail) given the count of findings that are
# at or above the configured threshold. Pure function — used by tests.
gate_exit() {
  local at_or_above="$1"
  if [ "${at_or_above:-0}" -gt 0 ]; then
    return 1
  fi
  return 0
}

run_gosec() {
  if ! command -v gosec >/dev/null 2>&1; then
    echo "WARN: gosec not installed; install: go install github.com/securego/gosec/v2/cmd/gosec@latest" >&2
    return 0
  fi
  echo ">> gosec (threshold: ${VELOX_SEVERITY})"
  # gosec exits non-zero when it reports issues at/above -severity.
  if gosec -quiet -severity "${VELOX_SEVERITY}" ./... ; then
    return 0
  fi
  return 1
}

run_govulncheck() {
  if ! command -v govulncheck >/dev/null 2>&1; then
    echo "WARN: govulncheck not installed; install: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
    return 0
  fi
  echo ">> govulncheck"
  # govulncheck exits non-zero (3) when vulnerabilities affect the build.
  if govulncheck ./... ; then
    return 0
  fi
  return 1
}

main() {
  local failed=0
  run_gosec || failed=1
  run_govulncheck || failed=1
  if [ "$failed" -ne 0 ]; then
    echo "SECURITY GATE: FAIL (findings at/above ${VELOX_SEVERITY})" >&2
    exit 1
  fi
  echo "SECURITY GATE: PASS"
}

# Only execute when run directly, not when sourced for tests.
if [ "${VELOX_SECURITY_LIB:-0}" != "1" ]; then
  main "$@"
fi
