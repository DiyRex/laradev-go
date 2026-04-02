#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# LaraDev installer / updater — macOS and Linux
#
# Usage (fresh install or update):
#   curl -fsSL https://raw.githubusercontent.com/DiyRex/laradev-go/main/scripts/install.sh | bash
#
# The script:
#   • Detects your OS and CPU architecture automatically
#   • Downloads laradev and mkcert as self-contained binaries (no package manager)
#   • Trusts the mkcert local CA so .test certificates work in browsers
#   • On UPDATE: replaces only the binary — never touches ~/.laradev/ (your
#     proxy configs and certificates are always preserved)
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

REPO="DiyRex/laradev-go"
MKCERT_REPO="FiloSottile/mkcert"
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
# Usage: download <url> <dest_file>
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
# Usage: github_latest <owner/repo>
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

  # Check if already installed.
  local existing_path existing_label
  existing_path="$(command -v "$BINARY" 2>/dev/null || true)"
  if [[ -n "$existing_path" ]]; then
    existing_label="update"
    info "Found existing installation at ${existing_path} — updating binary only"
    info "(~/.laradev/ configs and certificates will NOT be touched)"
  else
    existing_label="install"
    info "No existing installation found — fresh install"
  fi

  info "Fetching latest release tag…"
  local tag
  tag="$(github_latest "$REPO")"
  ok "Latest version: ${tag}"

  # Goreleaser outputs raw binaries named: laradev-{os}-{arch}
  local filename="${BINARY}-${os}-${arch}"
  local url="https://github.com/${REPO}/releases/download/${tag}/${filename}"
  local tmp_file
  tmp_file="$(mktemp)"

  info "Downloading ${filename} (${tag})…"
  if ! download "$url" "$tmp_file"; then
    rm -f "$tmp_file"
    die "Download failed. URL: ${url}"
  fi

  chmod +x "$tmp_file"

  # Verify it looks like an ELF/Mach-O binary, not an HTML error page.
  local first_bytes
  first_bytes="$(head -c 4 "$tmp_file" 2>/dev/null || true)"
  if echo "$first_bytes" | grep -q "<html\|<!DOC\|<HTML"; then
    rm -f "$tmp_file"
    die "Download returned an HTML page instead of a binary. The release ${tag} may not have a ${os}-${arch} build yet."
  fi

  # Move to install dir (use sudo if needed).
  local dest="${INSTALL_DIR}/${BINARY}"
  if [[ -w "$INSTALL_DIR" ]]; then
    mv "$tmp_file" "$dest"
  else
    info "Installing to ${INSTALL_DIR} (sudo required)…"
    sudo mv "$tmp_file" "$dest"
  fi

  ok "laradev ${existing_label}d → ${dest}"
}

# ─── Install mkcert (no package manager — direct binary from GitHub) ──────────
install_mkcert() {
  local os="$1" arch="$2"

  step "mkcert"

  # Skip if already installed.
  if command -v mkcert &>/dev/null; then
    local existing_ver
    existing_ver="$(mkcert --version 2>/dev/null || echo 'unknown')"
    ok "mkcert already installed (${existing_ver}) — skipping download"
    return 0
  fi

  info "Fetching latest mkcert release tag…"
  local tag
  tag="$(github_latest "$MKCERT_REPO")"
  ok "Latest mkcert: ${tag}"

  # mkcert release asset naming: mkcert-{tag}-{os}-{arch}
  # darwin arm64 is published as "darwin-arm64" from v1.4.4 onward.
  local filename="mkcert-${tag}-${os}-${arch}"
  local url="https://github.com/${MKCERT_REPO}/releases/download/${tag}/${filename}"
  local tmp_file
  tmp_file="$(mktemp)"

  info "Downloading ${filename}…"
  if ! download "$url" "$tmp_file"; then
    rm -f "$tmp_file"
    die "mkcert download failed. URL: ${url}"
  fi

  chmod +x "$tmp_file"

  local dest="${INSTALL_DIR}/mkcert"
  if [[ -w "$INSTALL_DIR" ]]; then
    mv "$tmp_file" "$dest"
  else
    info "Installing mkcert to ${INSTALL_DIR} (sudo required)…"
    sudo mv "$tmp_file" "$dest"
  fi

  ok "mkcert installed → ${dest}"

  # On Linux, mkcert needs nss-tools (libnss3-tools) for Firefox/Chromium support.
  # This is optional — mkcert -install still works without it (just skips NSS).
  if [[ "$os" == "linux" ]]; then
    info "Attempting to install nss-tools for browser certificate trust (optional)…"
    if command -v apt-get &>/dev/null; then
      sudo apt-get install -y -q libnss3-tools 2>/dev/null && ok "libnss3-tools installed" || \
        warn "Could not install libnss3-tools — certificates will still work in most browsers"
    elif command -v dnf &>/dev/null; then
      sudo dnf install -y nss-tools 2>/dev/null && ok "nss-tools installed" || \
        warn "Could not install nss-tools — skipping"
    elif command -v pacman &>/dev/null; then
      sudo pacman -S --noconfirm nss 2>/dev/null && ok "nss installed" || \
        warn "Could not install nss — skipping"
    else
      warn "Could not detect package manager — skipping nss-tools (optional)"
    fi
  fi
}

# ─── Trust the local CA ───────────────────────────────────────────────────────
install_local_ca() {
  step "Local Certificate Authority"

  if ! command -v mkcert &>/dev/null; then
    warn "mkcert not found — skipping CA trust (run 'mkcert -install' manually after installing mkcert)"
    return 0
  fi

  info "Running 'mkcert -install' to trust the local CA in your system keychain…"
  info "(This may ask for your sudo/admin password — required only once)"
  mkcert -install
  ok "Local CA trusted — browsers will accept .test certificates without warnings"
}

# ─── Create ~/.laradev directory structure ────────────────────────────────────
ensure_laradev_home() {
  step "~/.laradev"

  # These directories store proxy configs and certificates.
  # We NEVER remove or overwrite existing content here.
  local created=false
  for dir in "${LARADEV_HOME}/certs" "${LARADEV_HOME}/projects"; do
    if [[ ! -d "$dir" ]]; then
      mkdir -p "$dir"
      created=true
    fi
  done

  if $created; then
    ok "Created ${LARADEV_HOME}/"
  else
    ok "${LARADEV_HOME}/ already exists — configs and certificates preserved"
  fi
}

# ─── PATH check ───────────────────────────────────────────────────────────────
check_path() {
  if ! command -v laradev &>/dev/null; then
    echo ""
    warn "${INSTALL_DIR} is not in your PATH."
    echo ""
    echo "    Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo ""
    echo "      export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
    echo "    Then reload: source ~/.bashrc  (or open a new terminal)"
  fi
}

# ─── Main ─────────────────────────────────────────────────────────────────────
main() {
  echo ""
  printf "${BOLD}${CYAN}  LaraDev Installer${RST}\n"
  printf "  ${DIM}https://github.com/${REPO}${RST}\n"
  echo ""

  local platform os arch
  platform="$(detect_platform)"
  os="${platform%%/*}"
  arch="${platform##*/}"

  dim "Platform: ${os}/${arch}"

  install_laradev  "$os" "$arch"
  install_mkcert   "$os" "$arch"
  install_local_ca
  ensure_laradev_home
  check_path

  step "Done"
  echo ""
  echo "  Next steps:"
  echo ""
  printf "  ${CYAN}1.${RST}  cd into a Laravel project\n"
  printf "  ${CYAN}2.${RST}  ${BOLD}laradev${RST}               — open interactive TUI\n"
  printf "  ${CYAN}3.${RST}  ${BOLD}laradev proxy:setup${RST}   — set up HTTPS .test domain (optional)\n"
  printf "  ${CYAN}4.${RST}  ${BOLD}laradev up${RST}            — start all services\n"
  echo ""
  printf "  Run ${BOLD}laradev help${RST} for all commands.\n"
  echo ""
}

main "$@"
