#!/usr/bin/env bash
# backup.sh - copy the Naraberu database into data/backups/
# Usage (Git Bash): ./scripts/backup.sh
# Optional: DATA_DIR=/path/to/data ./scripts/backup.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DATA_DIR="${DATA_DIR:-$ROOT/data}"
DB_PATH="$DATA_DIR/naraberu.db"
BACKUP_DIR="$DATA_DIR/backups"

if [ ! -f "$DB_PATH" ]; then
  echo "error: database not found at $DB_PATH" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/naraberu_$TIMESTAMP.db"
cp "$DB_PATH" "$BACKUP_FILE"
echo "Backup created: $BACKUP_FILE"

cd "$BACKUP_DIR"
ls -t naraberu_*.db 2>/dev/null | tail -n +6 | xargs rm -f 2>/dev/null || true
echo "Backups remaining:"
ls -la naraberu_*.db 2>/dev/null || true
