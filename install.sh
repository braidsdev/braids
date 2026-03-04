#!/bin/sh
set -eu

REPO="braidsdev/braids"
INSTALL_DIR="/usr/local/bin"
FALLBACK_DIR="$HOME/.braids/bin"

main() {
    detect_platform
    fetch_latest_version
    download_and_verify
    install_binary
    print_success
}

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux)  OS="linux" ;;
        Darwin) OS="darwin" ;;
        *)      err "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        amd64)   ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        arm64)   ARCH="arm64" ;;
        *)       err "Unsupported architecture: $ARCH" ;;
    esac

    log "Detected platform: ${OS}/${ARCH}"
}

fetch_latest_version() {
    log "Fetching latest release..."
    VERSION="$(curl -sf "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' \
        | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')" \
        || err "Failed to fetch latest release. Check your internet connection."

    if [ -z "$VERSION" ]; then
        err "Could not determine latest version."
    fi

    log "Latest version: ${VERSION}"
}

download_and_verify() {
    ARCHIVE="braids_${OS}_${ARCH}.tar.gz"
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    ARCHIVE_URL="${BASE_URL}/${ARCHIVE}"
    CHECKSUMS_URL="${BASE_URL}/checksums.txt"

    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "$TMPDIR"' EXIT

    log "Downloading ${ARCHIVE}..."
    curl -sfL "$ARCHIVE_URL" -o "${TMPDIR}/${ARCHIVE}" \
        || err "Failed to download ${ARCHIVE_URL}"

    log "Verifying checksum..."
    curl -sfL "$CHECKSUMS_URL" -o "${TMPDIR}/checksums.txt" \
        || err "Failed to download checksums."

    EXPECTED="$(grep "${ARCHIVE}" "${TMPDIR}/checksums.txt" | awk '{print $1}')"
    if [ -z "$EXPECTED" ]; then
        err "No checksum found for ${ARCHIVE} in checksums.txt"
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL="$(sha256sum "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        ACTUAL="$(shasum -a 256 "${TMPDIR}/${ARCHIVE}" | awk '{print $1}')"
    else
        log "Warning: no sha256 tool found, skipping checksum verification."
        ACTUAL="$EXPECTED"
    fi

    if [ "$ACTUAL" != "$EXPECTED" ]; then
        err "Checksum mismatch!\n  Expected: ${EXPECTED}\n  Actual:   ${ACTUAL}"
    fi

    log "Checksum verified."

    tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"
}

install_binary() {
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMPDIR}/braids" "${INSTALL_DIR}/braids"
        log "Installed to ${INSTALL_DIR}/braids"
    elif command -v sudo >/dev/null 2>&1; then
        log "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "${TMPDIR}/braids" "${INSTALL_DIR}/braids"
        log "Installed to ${INSTALL_DIR}/braids"
    else
        log "${INSTALL_DIR} is not writable, falling back to ${FALLBACK_DIR}"
        mkdir -p "$FALLBACK_DIR"
        mv "${TMPDIR}/braids" "${FALLBACK_DIR}/braids"
        INSTALL_DIR="$FALLBACK_DIR"
        log "Installed to ${FALLBACK_DIR}/braids"
        if ! echo "$PATH" | tr ':' '\n' | grep -qx "$FALLBACK_DIR"; then
            log ""
            log "Add Braids to your PATH by adding this to your shell profile:"
            log "  export PATH=\"${FALLBACK_DIR}:\$PATH\""
        fi
    fi

    chmod +x "${INSTALL_DIR}/braids"
}

print_success() {
    log ""
    log "Braids ${VERSION} installed successfully!"
    log ""
    log "Get started:"
    log "  braids init"
    log "  braids serve"
    log ""
    log "Docs: https://braids.dev/docs"
}

log() {
    printf '%s\n' "$@"
}

err() {
    log "Error: $1" >&2
    exit 1
}

main
