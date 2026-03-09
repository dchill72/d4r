#!/usr/bin/env sh
set -eu

REPO="${D4R_REPO:-dchill72/d4r}"
BIN_NAME="d4r"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${D4R_VERSION:-latest}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

detect_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    linux) echo "linux" ;;
    darwin) echo "darwin" ;;
    *)
      echo "error: unsupported OS: $os (supported: linux, darwin)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "error: unsupported architecture: $arch (supported: amd64, arm64)" >&2
      exit 1
      ;;
  esac
}

download() {
  url="$1"
  out="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$out" "$url"
  else
    echo "error: curl or wget is required to download files" >&2
    exit 1
  fi
}

resolve_version() {
  if [ "$VERSION" != "latest" ]; then
    echo "$VERSION"
    return
  fi
  if ! command -v curl >/dev/null 2>&1; then
    echo "error: curl is required for latest version discovery; set D4R_VERSION=vX.Y.Z to skip API lookup" >&2
    exit 1
  fi

  api_url="https://api.github.com/repos/$REPO/releases/latest"
  version="$(curl -fsSL "$api_url" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$version" ]; then
    echo "error: could not determine latest release from $api_url" >&2
    exit 1
  fi
  echo "$version"
}

verify_checksum() {
  tar_file="$1"
  checksums_file="$2"
  need_cmd shasum

  expected="$(awk -v file="$(basename "$tar_file")" '$2 == file {print $1}' "$checksums_file")"
  if [ -z "$expected" ]; then
    echo "error: checksum entry for $(basename "$tar_file") not found" >&2
    exit 1
  fi

  actual="$(shasum -a 256 "$tar_file" | awk '{print $1}')"
  if [ "$expected" != "$actual" ]; then
    echo "error: checksum mismatch for $(basename "$tar_file")" >&2
    exit 1
  fi
}

install_binary() {
  src="$1"
  dst="$2/$BIN_NAME"

  if [ ! -d "$2" ]; then
    mkdir -p "$2"
  fi

  if [ -w "$2" ]; then
    install -m 0755 "$src" "$dst"
  else
    echo "install dir $2 is not writable, trying sudo..."
    sudo install -m 0755 "$src" "$dst"
  fi
}

main() {
  need_cmd tar
  need_cmd awk
  need_cmd sed
  need_cmd install
  need_cmd uname

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"

  base_url="https://github.com/$REPO/releases/download/$version"
  archive="${BIN_NAME}_${version}_${os}_${arch}.tar.gz"
  checksums="checksums.txt"

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT INT TERM

  archive_path="$tmp_dir/$archive"
  checksums_path="$tmp_dir/$checksums"

  echo "Downloading $archive..."
  download "$base_url/$archive" "$archive_path"

  echo "Downloading $checksums..."
  download "$base_url/$checksums" "$checksums_path"

  echo "Verifying checksum..."
  verify_checksum "$archive_path" "$checksums_path"

  echo "Extracting..."
  tar -xzf "$archive_path" -C "$tmp_dir"

  if [ ! -f "$tmp_dir/$BIN_NAME" ]; then
    echo "error: binary $BIN_NAME not found in archive" >&2
    exit 1
  fi

  echo "Installing to $INSTALL_DIR..."
  install_binary "$tmp_dir/$BIN_NAME" "$INSTALL_DIR"

  echo "Installed $BIN_NAME $version to $INSTALL_DIR/$BIN_NAME"
  echo "Run: $BIN_NAME"
}

main "$@"
