#!/usr/bin/env bash
#
# Cross-compile static velox binaries (CGO disabled) for all supported targets.
# Output: dist/velox-<os>-<arch>[.exe]
set -euo pipefail

VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo 0.1.0-dev)}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
DATE="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
PKG="github.com/koller-nexus/velox/internal/version"
LDFLAGS="-s -w -X ${PKG}.Version=${VERSION} -X ${PKG}.Commit=${COMMIT} -X ${PKG}.Date=${DATE}"

TARGETS=(
  "linux/amd64" "linux/arm64"
  "darwin/amd64" "darwin/arm64"
  "windows/amd64" "windows/arm64"
)

mkdir -p dist
for t in "${TARGETS[@]}"; do
  os="${t%/*}"; arch="${t#*/}"
  out="dist/velox-${os}-${arch}"
  [ "$os" = "windows" ] && out="${out}.exe"
  echo ">> building ${out}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
    go build -ldflags "$LDFLAGS" -o "$out" ./cmd/velox
done
echo "done: $(ls dist)"
