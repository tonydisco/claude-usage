#!/usr/bin/env bash
#
# claude-usage installer
#
#   curl -fsSL https://raw.githubusercontent.com/tonydisco/claude-usage/main/install.sh | bash
#
# Downloads the latest release archive for the current OS/arch from GitHub,
# verifies the checksum, and installs the binary to $PREFIX/bin (default
# /usr/local on macOS+Linux, falls back to ~/.local for unprivileged installs).

set -euo pipefail

REPO="tonydisco/claude-usage"
BIN="claude-usage"

err() { printf "\033[31merror:\033[0m %s\n" "$*" >&2; exit 1; }
log() { printf "\033[36m=>\033[0m %s\n" "$*"; }

detect_os() {
  case "$(uname -s)" in
    Darwin) echo "Darwin" ;;
    Linux)  echo "Linux" ;;
    *)      err "unsupported OS: $(uname -s) — install from source: go install github.com/${REPO}/cmd/${BIN}@latest" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "x86_64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) err "unsupported arch: $(uname -m)" ;;
  esac
}

pick_prefix() {
  if [[ -n "${PREFIX:-}" ]]; then echo "$PREFIX"; return; fi
  if [[ -w /usr/local/bin ]] 2>/dev/null; then echo "/usr/local"; return; fi
  if command -v sudo >/dev/null 2>&1 && [[ "${USE_SUDO:-1}" == "1" ]]; then
    echo "/usr/local"; return
  fi
  echo "$HOME/.local"
}

main() {
  command -v curl >/dev/null 2>&1 || err "curl is required"
  command -v tar  >/dev/null 2>&1 || err "tar is required"

  local os arch prefix tmp tag asset url
  os=$(detect_os)
  arch=$(detect_arch)
  prefix=$(pick_prefix)
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  log "Looking up latest release of ${REPO}…"
  tag=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  [[ -n "$tag" ]] || err "could not determine latest release tag"
  log "Latest: $tag"

  asset="${BIN}_${tag#v}_${os}_${arch}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${tag}/${asset}"

  log "Downloading $asset"
  curl -fsSL -o "$tmp/$asset" "$url" || err "download failed: $url"

  log "Verifying checksum"
  curl -fsSL -o "$tmp/checksums.txt" "https://github.com/${REPO}/releases/download/${tag}/checksums.txt"
  (cd "$tmp" && grep " $asset\$" checksums.txt | shasum -a 256 -c -) \
    || err "checksum mismatch"

  log "Extracting"
  tar -xzf "$tmp/$asset" -C "$tmp"

  local dest="$prefix/bin"
  mkdir -p "$dest"
  if [[ -w "$dest" ]]; then
    install -m 0755 "$tmp/$BIN" "$dest/$BIN"
  else
    log "Installing to $dest (requires sudo)"
    sudo install -m 0755 "$tmp/$BIN" "$dest/$BIN"
  fi

  log "Installed: $dest/$BIN"
  if ! command -v "$BIN" >/dev/null 2>&1; then
    log "Note: $dest is not on \$PATH. Add this to your shell rc:"
    printf '  export PATH="%s:$PATH"\n' "$dest"
  else
    "$BIN" --version || true
  fi
  log "Next: run \`${BIN} login\` and paste your claude.ai sessionKey cookie."
}

main "$@"
