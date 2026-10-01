#!/usr/bin/env bash
# One-shot production build: Windows icon/manifest resources + GUI exe.
# Usage (Git Bash, from anywhere):
#   ./scripts/build-production.sh
# Env:
#   ARCH=amd64|arm64   (default amd64)
#   OUT=path/to/exe    (default build/bin/naraberu.exe)
#   VERSION=x.y.z      (default 1.0.0) Windows version resource
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ARCH="${ARCH:-amd64}"
OUT="${OUT:-build/bin/naraberu.exe}"
VERSION="${VERSION:-1.0.0}"
SYSO="wails_windows.syso"

if ! command -v wails >/dev/null 2>&1 && ! command -v wails3 >/dev/null 2>&1; then
  echo "error: wails/wails3 CLI not found (need Wails v3)" >&2
  exit 1
fi
WAILS="$(command -v wails3 2>/dev/null || command -v wails)"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go not found" >&2
  exit 1
fi

if [[ ! -f build/windows/icon.ico ]]; then
  echo "==> generating icon.ico from appicon.png"
  (cd build && "$WAILS" generate icons -input appicon.png)
fi

for f in build/windows/icon.ico build/windows/wails.exe.manifest build/windows/info.json; do
  if [[ ! -f "$f" ]]; then
    echo "error: missing $f" >&2
    exit 1
  fi
done

cleanup() {
  rm -f "$ROOT/$SYSO"
}
trap cleanup EXIT

# Concrete version strings for the exe Properties dialog (no template placeholders).
INFO_JSON="$ROOT/build/windows/info.generated.json"
file_ver="${VERSION}"
case "$file_ver" in
  *.*.*.*) ;;
  *.*.*.) ;;
  *.*.*) file_ver="${VERSION}.0" ;;
  *.*) file_ver="${VERSION}.0.0" ;;
  *) file_ver="${VERSION}.0.0.0" ;;
esac

cat > "$INFO_JSON" <<EOF
{
	"fixed": {
		"file_version": "$file_ver"
	},
	"info": {
		"0000": {
			"ProductVersion": "$VERSION",
			"CompanyName": "Naraberu contributors",
			"FileDescription": "Naraberu",
			"LegalCopyright": "Copyright Naraberu contributors",
			"ProductName": "Naraberu",
			"Comments": "Offline anime series tracker"
		}
	}
}
EOF

echo "==> generating Windows resources ($ARCH) version $VERSION"
(
  cd build/windows
  "$WAILS" generate syso \
    -arch "$ARCH" \
    -icon icon.ico \
    -manifest wails.exe.manifest \
    -info info.generated.json \
    -out "../../$SYSO"
)

echo "==> building production exe → $OUT"
mkdir -p "$(dirname "$OUT")"
go build -tags production -trimpath -ldflags="-w -s -H windowsgui" -o "$OUT" .

echo "==> done: $OUT (version $VERSION)"
