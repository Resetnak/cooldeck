#!/bin/sh
# Install the latest cooldeck release.
#
#   curl -fsSL https://raw.githubusercontent.com/Resetnak/cooldeck/main/install.sh | sh
#
# Reads nothing, writes one binary, and verifies its checksum before doing so.
# Override the destination with COOLDECK_INSTALL_DIR, or pin a version with
# COOLDECK_VERSION=v0.1.1.
#
# POSIX sh on purpose: this has to run on a stock Alpine container as readily
# as on a Mac.
set -eu

REPO="Resetnak/cooldeck"
INSTALL_DIR="${COOLDECK_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	echo "install: $1" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || fail "$1 is required but not installed"
}

need uname
need tar
need mktemp

# curl or wget, whichever the machine happens to have.
if command -v curl >/dev/null 2>&1; then
	fetch() { curl -fsSL "$1" -o "$2"; }
	fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
	fetch() { wget -qO "$2" "$1"; }
	fetch_stdout() { wget -qO- "$1"; }
else
	fail "neither curl nor wget is installed"
fi

case "$(uname -s)" in
Linux) os="Linux" ;;
Darwin) os="Darwin" ;;
*) fail "unsupported operating system: $(uname -s). Windows users: download the .zip from https://github.com/$REPO/releases/latest" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch="x86_64" ;;
arm64 | aarch64) arch="arm64" ;;
*) fail "unsupported architecture: $(uname -m)" ;;
esac

version="${COOLDECK_VERSION:-}"
if [ -z "$version" ]; then
	# Follow the /latest redirect rather than parsing the API, so this needs no
	# JSON tooling and no API token.
	version=$(fetch_stdout "https://github.com/$REPO/releases/latest" 2>/dev/null |
		sed -n 's|.*/releases/tag/\(v[0-9][^"]*\)".*|\1|p' | head -n 1)
	[ -n "$version" ] || fail "could not determine the latest version; set COOLDECK_VERSION"
fi

archive="cooldeck_${version#v}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$version"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "install: downloading cooldeck $version for $os/$arch"
fetch "$base/$archive" "$tmp/$archive" || fail "download failed: $base/$archive"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "could not download checksums.txt"

# Verify before trusting. A download that cannot be checked is not installed.
expected=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$expected" ] || fail "$archive is not listed in checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/$archive" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
	actual=$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)
else
	fail "neither sha256sum nor shasum is installed, so the download cannot be verified"
fi
[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive: expected $expected, got $actual"

tar -xzf "$tmp/$archive" -C "$tmp" cooldeck
mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/cooldeck" "$INSTALL_DIR/cooldeck" 2>/dev/null ||
	{ cp "$tmp/cooldeck" "$INSTALL_DIR/cooldeck" && chmod 0755 "$INSTALL_DIR/cooldeck"; }

echo "install: cooldeck $version -> $INSTALL_DIR/cooldeck"
case ":$PATH:" in
*":$INSTALL_DIR:"*) echo "install: run 'cooldeck --demo' to try it offline" ;;
*) echo "install: $INSTALL_DIR is not on your PATH; add it, or run $INSTALL_DIR/cooldeck" ;;
esac
