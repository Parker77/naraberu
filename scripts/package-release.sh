#!/usr/bin/env bash
# package-release.sh - build a Windows release zip for GitHub Releases
# Usage (Git Bash):
#   ./scripts/package-release.sh
#   VERSION=1.2.0 ./scripts/package-release.sh
#
# Code signing is not used for this project. Windows SmartScreen may warn;
# see README “Getting started”.
#
# Output: dist/Naraberu-<version>-windows-amd64.zip
# Contents: naraberu.exe, LICENSE, README.md, docs/
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-1.0.0}"
ARCH="${ARCH:-amd64}"
OUT_EXE="${OUT_EXE:-build/bin/naraberu.exe}"
DIST="$ROOT/dist"
NAME="Naraberu-${VERSION}-windows-${ARCH}"
STAGE="$DIST/$NAME"

echo "==> building production binary (version $VERSION)"
VERSION="$VERSION" ARCH="$ARCH" OUT="$OUT_EXE" ./scripts/build-production.sh

if [[ ! -f "$OUT_EXE" ]]; then
  echo "error: missing $OUT_EXE" >&2
  exit 1
fi

echo "==> staging $STAGE"
rm -rf "$STAGE"
mkdir -p "$STAGE/docs"
cp "$OUT_EXE" "$STAGE/naraberu.exe"
cp LICENSE README.md "$STAGE/"
cp docs/*.md docs/anime-list.example.txt "$STAGE/docs/" 2>/dev/null || true

echo "==> zipping"
mkdir -p "$DIST"
ZIP="$DIST/${NAME}.zip"
rm -f "$ZIP"
# Windows zip via PowerShell (Compress-Archive). Use real Windows paths.
WIN_STAGE="$(cygpath -w "$STAGE" 2>/dev/null || echo "$STAGE")"
WIN_ZIP="$(cygpath -w "$ZIP" 2>/dev/null || echo "$ZIP")"
powershell -NoProfile -Command "Compress-Archive -Path '$WIN_STAGE' -DestinationPath '$WIN_ZIP' -Force"

echo "==> done: $ZIP"
ls -la "$ZIP"
