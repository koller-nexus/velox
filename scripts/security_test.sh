#!/usr/bin/env bash
#
# Tests for scripts/security.sh gate logic (T038). Sources the script in library
# mode and asserts the threshold/exit mapping without running real scanners.
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VELOX_SECURITY_LIB=1 source "${DIR}/security.sh"

fail=0
assert_eq() { # want got msg
  if [ "$1" != "$2" ]; then
    echo "FAIL: $3 (want '$1', got '$2')" >&2
    fail=1
  fi
}

# severity_rank ordering
assert_eq 1 "$(severity_rank low)" "low rank"
assert_eq 2 "$(severity_rank MEDIUM)" "medium rank (case-insensitive)"
assert_eq 3 "$(severity_rank high)" "high rank"
assert_eq 4 "$(severity_rank Critical)" "critical rank"
assert_eq 0 "$(severity_rank bogus)" "unknown rank"

# gate_exit: 0 findings -> pass (0); >0 -> fail (1)
if gate_exit 0; then :; else echo "FAIL: gate_exit 0 should pass" >&2; fail=1; fi
if gate_exit 2; then echo "FAIL: gate_exit 2 should fail" >&2; fail=1; fi

if [ "$fail" -eq 0 ]; then
  echo "security_test.sh: PASS"
else
  echo "security_test.sh: FAIL" >&2
  exit 1
fi
