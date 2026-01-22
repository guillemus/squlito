#!/usr/bin/env bash

set -euo pipefail

REPO="guillemus/squlito"
BINARY="squlito"
PREFIX="${PREFIX:-/usr/local}"
VERSION="${VERSION:-}"

if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required" >&2
    exit 1
fi

if ! command -v tar >/dev/null 2>&1; then
    echo "tar is required" >&2
    exit 1
fi

if ! command -v install >/dev/null 2>&1; then
    echo "install is required" >&2
    exit 1
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${OS}" in
    darwin|linux) ;;
    *)
        echo "unsupported OS: ${OS}" >&2
        exit 1
        ;;
esac

case "${ARCH}" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)
        echo "unsupported arch: ${ARCH}" >&2
        exit 1
        ;;
esac

if [ -z "${VERSION}" ]; then
    VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | awk -F '"' '/tag_name/ {print $4; exit}')"
fi

if [ -z "${VERSION}" ]; then
    echo "failed to resolve version" >&2
    exit 1
fi

VERSION_CLEAN="${VERSION#v}"
ARCHIVE_NAME="${BINARY}_${VERSION_CLEAN}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

temp_dir="$(mktemp -d)"
cleanup() {
    rm -rf "${temp_dir}"
}
trap cleanup EXIT

curl -fsSL "${DOWNLOAD_URL}" -o "${temp_dir}/${ARCHIVE_NAME}"
tar -xzf "${temp_dir}/${ARCHIVE_NAME}" -C "${temp_dir}"

install -m 755 "${temp_dir}/${BINARY}" "${PREFIX}/bin/${BINARY}"

echo "installed ${BINARY} to ${PREFIX}/bin/${BINARY}"
