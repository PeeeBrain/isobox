#!/usr/bin/env bash
set -euo pipefail

REPO="PeeeBrain/isobox"
BIN_NAME="isobox"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "error: $1 is required" >&2
    exit 1
  }
}

need curl
need tar
need sha256sum

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux) ;;
  *)
    echo "error: isobox currently supports Linux and WSL2." >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *)
    echo "error: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="${BIN_NAME}_${os}_${arch}.tar.gz"
base_url="https://github.com/${REPO}/releases/latest/download"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading ${asset}..."
curl -fsSL "${base_url}/${asset}" -o "${tmp_dir}/${asset}"
curl -fsSL "${base_url}/checksums.txt" -o "${tmp_dir}/checksums.txt"

echo "Verifying checksum..."
(
  cd "$tmp_dir"
  grep "  ${asset}$" checksums.txt | sha256sum -c -
)

mkdir -p "$INSTALL_DIR"

tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"

if [ ! -f "${tmp_dir}/${BIN_NAME}" ]; then
  echo "error: ${BIN_NAME} binary not found in archive" >&2
  exit 1
fi

install -m 0755 "${tmp_dir}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"

echo "Installed ${BIN_NAME} to ${INSTALL_DIR}/${BIN_NAME}"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo
    echo "Note: ${INSTALL_DIR} is not on your PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo
"${INSTALL_DIR}/${BIN_NAME}" --help >/dev/null 2>&1 || true
echo "Done."
