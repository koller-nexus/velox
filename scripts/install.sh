#!/bin/sh
# Instalador do velox para macOS e Linux
# Uso: curl -fsSL https://raw.githubusercontent.com/koller-nexus/velox/main/scripts/install.sh | sh
set -eu

REPO="koller-nexus/velox"
BINARY="velox"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# --- Detecta OS ---
OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  Linux)  OS="linux" ;;
  *)
    echo "Erro: sistema operacional não suportado: $OS" >&2
    echo "Para Windows, baixe o .zip em https://github.com/$REPO/releases" >&2
    exit 1
    ;;
esac

# --- Detecta arquitetura ---
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Erro: arquitetura não suportada: $ARCH" >&2
    exit 1
    ;;
esac

# --- Resolve a versão (última release por padrão) ---
VERSION="${VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
    | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  if [ -z "$VERSION" ]; then
    echo "Erro: não foi possível determinar a última versão" >&2
    exit 1
  fi
fi

VERSION_NUM="${VERSION#v}"
ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"
CHECKSUMS_URL="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"

echo "Instalando $BINARY $VERSION ($OS/$ARCH)..."

# --- Download em diretório temporário ---
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$URL" -o "$TMP_DIR/$ARCHIVE"
curl -fsSL "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt"

# --- Verifica checksum SHA256 ---
cd "$TMP_DIR"
EXPECTED="$(grep "$ARCHIVE" checksums.txt | awk '{print $1}')"
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Erro: checksum inválido! Esperado: $EXPECTED, obtido: $ACTUAL" >&2
  exit 1
fi
echo "Checksum verificado ✓"

# --- Extrai e instala ---
tar -xzf "$ARCHIVE"

if [ -w "$INSTALL_DIR" ]; then
  mv "$BINARY" "$INSTALL_DIR/$BINARY"
else
  echo "Sem permissão de escrita em $INSTALL_DIR, usando sudo..."
  sudo mv "$BINARY" "$INSTALL_DIR/$BINARY"
fi

chmod +x "$INSTALL_DIR/$BINARY"

echo ""
echo "✅ $BINARY instalado em $INSTALL_DIR/$BINARY"
"$INSTALL_DIR/$BINARY" version || true
