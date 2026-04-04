#!/usr/bin/env sh

set -eu

REPO="${REPO:-raoooool/qiao}"
PROJECT_NAME="${PROJECT_NAME:-qiao}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-}"

say() {
  printf '%s\n' "$*"
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

detect_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux|darwin)
      printf '%s\n' "$os"
      ;;
    *)
      die "unsupported operating system: $os"
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)
      printf 'amd64\n'
      ;;
    arm64|aarch64)
      printf 'arm64\n'
      ;;
    *)
      die "unsupported architecture: $arch"
      ;;
  esac
}

sha256_file() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl dgst -sha256 "$file" | awk '{print $NF}'
    return
  fi
  die "missing checksum tool: install sha256sum, shasum, or openssl"
}

latest_version() {
  response="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")" || die "failed to query latest release"
  version="$(printf '%s' "$response" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$version" ] || die "could not determine latest release tag"
  printf '%s\n' "$version"
}

ensure_path_hint() {
  case ":$PATH:" in
    *":$INSTALL_DIR:"*)
      ;;
    *)
      say "Add ${INSTALL_DIR} to your PATH to use ${PROJECT_NAME} directly."
      ;;
  esac
}

need_cmd curl
need_cmd tar
need_cmd awk
need_cmd sed
need_cmd mktemp

os="$(detect_os)"
arch="$(detect_arch)"
version="${VERSION:-$(latest_version)}"
archive_name="${PROJECT_NAME}_${os}_${arch}.tar.gz"
checksum_name="${PROJECT_NAME}_checksums.txt"
release_base_url="https://github.com/${REPO}/releases/download/${version}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT HUP TERM

archive_path="${tmpdir}/${archive_name}"
checksum_path="${tmpdir}/${checksum_name}"
extract_dir="${tmpdir}/extract"

say "Downloading ${PROJECT_NAME} ${version} for ${os}/${arch}..."
curl -fsSL -o "$archive_path" "${release_base_url}/${archive_name}" || die "failed to download ${archive_name}"
curl -fsSL -o "$checksum_path" "${release_base_url}/${checksum_name}" || die "failed to download ${checksum_name}"

expected_checksum="$(awk -v file="$archive_name" '$2 == file { print $1 }' "$checksum_path")"
[ -n "$expected_checksum" ] || die "checksum entry not found for ${archive_name}"

actual_checksum="$(sha256_file "$archive_path")"
[ "$expected_checksum" = "$actual_checksum" ] || die "checksum mismatch for ${archive_name}"

mkdir -p "$extract_dir"
tar -xzf "$archive_path" -C "$extract_dir"

binary_path="$(find "$extract_dir" -type f -name "$PROJECT_NAME" | head -n 1)"
[ -n "$binary_path" ] || die "could not find ${PROJECT_NAME} binary in archive"

mkdir -p "$INSTALL_DIR"
cp "$binary_path" "${INSTALL_DIR}/${PROJECT_NAME}"
chmod 755 "${INSTALL_DIR}/${PROJECT_NAME}"

say "Installed ${PROJECT_NAME} to ${INSTALL_DIR}/${PROJECT_NAME}"
ensure_path_hint
