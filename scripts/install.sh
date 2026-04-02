#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# LaraDev installer / updater — macOS and Linux
#
# Usage (fresh install or update):
#   curl -fsSL https://raw.githubusercontent.com/DiyRex/laradev-go/main/scripts/install.sh | bash
#
# laradev is a fully self-contained binary — no dependencies, no package
# managers, nothing extra to install. This script only:
#   • Downloads the right binary for your OS and CPU architecture
#   • Places it in /usr/local/bin
#   • Creates ~/.laradev/ for proxy configs and certificates
#
# On UPDATE: replaces only the binary — ~/.laradev/ is never touched.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO="DiyRex/laradev-go"
BINARY="laradev"
INSTALL_DIR="/usr/local/bin"
LARADEV_HOME="${HOME}/.laradev"

# ─── colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; RST='\033[0m'

ok()    { printf "  ${GREEN}✓${RST}  %s\n" "$*"; }
fail()  { printf "  ${RED}✗${RST}  %s\n" "$*" >&2; }
warn()  { printf "  ${YELLOW}!${RST}  %s\n" "$*"; }
info()  { printf "  ${CYAN}→${RST}  %s\n" "$*"; }
step()  { printf "\n${BOLD}%s${RST}\n" "$*"; }
dim()   { printf "  ${DIM}%s${RST}\n" "$*"; }

die() { fail "$*"; exit 1; }

# ─── OS / arch detection ──────────────────────────────────────────────────────
detect_platform() {
  local os arch

  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux"  ;;
    *)      die "Unsupported OS: $(uname -s). Only macOS and Linux are supported." ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64)   arch="amd64" ;;
    arm64|aarch64)  arch="arm64" ;;
    *)               die "Unsupported CPU architecture: $(uname -m)" ;;
  esac

  echo "${os}/${arch}"
}

# ─── HTTP download helper ─────────────────────────────────────────────────────
download() {
  local url="$1" dest="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL --retry 3 --retry-delay 2 -o "$dest" "$url"
  elif command -v wget &>/dev/null; then
    wget -q --tries=3 -O "$dest" "$url"
  else
    die "Neither curl nor wget found. Install one and re-run."
  fi
}

# ─── GitHub latest release tag ────────────────────────────────────────────────
github_latest() {
  local repo="$1" tag=""
  local api_url="https://api.github.com/repos/${repo}/releases/latest"

  if command -v curl &>/dev/null; then
    tag=$(curl -fsSL --retry 3 "$api_url" 2>/dev/null \
      | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | head -1)
  elif command -v wget &>/dev/null; then
    tag=$(wget -qO- "$api_url" 2>/dev/null \
      | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/' | head -1)
  fi

  [[ -n "$tag" ]] || die "Could not fetch latest release for ${repo}. Check your internet connection."
  echo "$tag"
}

# ─── Install or update the laradev binary ─────────────────────────────────────
install_laradev() {
  local os="$1" arch="$2"

  step "LaraDev"

  local mode="Installing"
  command -v "$BINARY" &>/dev/null && mode="Updating"
  [[ "$mode" == "Updating" ]] && info "Existing installation found — replacing binary only (~/.laradev/ untouched)"

  info "Fetching latest release tag…"
  local tag
  tag="$(github_latest "$REPO")"
  ok "Latest version: ${tag}"

  # GoReleaser outputs raw binaries named: laradev-{os}-{arch}
  local filename="${BINARY}-${os}-${arch}"
  local url="https://github.com/${REPO}/releases/download/${tag}/${filename}"
  local tmp_file
  tmp_file="$(mktemp)"

  info "Downloading ${filename}…"
  if ! download "$url" "$tmp_file"; then
    rm -f "$tmp_file"
    die "Download failed. URL: ${url}"
  fi

  chmod +x "$tmp_file"

  # Sanity check — reject HTML error pages.
  if head -c 5 "$tmp_file" 2>/dev/null | grep -qi "^<html\|^<!doc"; then
    rm -f "$tmp_file"
    die "Got an HTML response instead of a binary. Release ${tag} may not have a ${os}-${arch} build yet."
  fi

  local dest="${INSTALL_DIR}/${BINARY}"
  if [[ -w "$INSTALL_DIR" ]]; then
    mv "$tmp_file" "$dest"
  else
    info "Installing to ${INSTALL_DIR} (sudo required)…"
    sudo mv "$tmp_file" "$dest"
  fi

  ok "laradev ${mode,,}d → ${dest}"
}

# ─── Create ~/.laradev directory structure ────────────────────────────────────
ensure_laradev_home() {
  step "~/.laradev"
  local created=false
  for dir in "${LARADEV_HOME}/certs" "${LARADEV_HOME}/projects" "${LARADEV_HOME}/ca"; do
    if [[ ! -d "$dir" ]]; then
      mkdir -p "$dir"
      created=true
    fi
  done
  $created && ok "Created ${LARADEV_HOME}/" || ok "${LARADEV_HOME}/ already exists — configs and certificates preserved"
}

# ─── PATH check ───────────────────────────────────────────────────────────────
check_path() {
  if ! command -v laradev &>/dev/null; then
    echo ""
    warn "${INSTALL_DIR} is not in your PATH."
    echo "    Add to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "      export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo "    Then reload: source ~/.bashrc  (or open a new terminal)"
  fi
}

# ─── Main ─────────────────────────────────────────────────────────────────────
main() {
  echo ""
  printf "${BOLD}${CYAN}  LaraDev Installer${RST}\n"
  printf "  ${DIM}Self-contained binary — no dependencies required${RST}\n"
  echo ""

  local platform os arch
  platform="$(detect_platform)"
  os="${platform%%/*}"
  arch="${platform##*/}"

  dim "Platform: ${os}/${arch}"

  install_laradev "$os" "$arch"
  ensure_laradev_home
  check_path

  step "Done"
  echo ""
  echo "  Next steps:"
  echo ""
  printf "  ${CYAN}1.${RST}  cd into a Laravel project\n"
  printf "  ${CYAN}2.${RST}  ${BOLD}laradev${RST}               — open interactive TUI\n"
  printf "  ${CYAN}3.${RST}  ${BOLD}laradev proxy:setup${RST}   — set up HTTPS .test domain (optional, one time)\n"
  printf "  ${CYAN}4.${RST}  ${BOLD}laradev up${RST}            — start all services\n"
  echo ""
  printf "  Run ${BOLD}laradev help${RST} for all commands.\n"
  echo ""
}

main "$@"
