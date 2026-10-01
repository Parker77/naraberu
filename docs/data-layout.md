# Data layout

Naraberu keeps all of its data in a single `data` folder. It does not use a machine-global config directory.

## Where is `data`?

1. If `NARABERU_DATA_DIR` is set, that path is used (absolute or relative).
2. Otherwise, a `data` folder is created **next to the executable**.
3. If that location is not writable (for example a Program Files install), the app falls back to `./data` under the working directory.
4. When running via `go run`, the temporary build binary is ignored and `./data` under the project directory is used (or `NARABERU_DATA_DIR`).

You can move an install by copying the whole app folder (exe + `data/`).

## Contents

| Path | Purpose |
|------|---------|
| `data/naraberu.db` | SQLite database (series, settings) |
| `data/thumbnails/` | Local thumbnail images |
| `data/backups/` | Startup backups (last 5) |
| `data/sounds/` | Extra click sounds (mp3) you add yourself |
| `data/webview/` | WebView2 browser profile |

## Environment variable

```bash
# Git Bash
NARABERU_DATA_DIR=/path/to/my-naraberu-data ./naraberu.exe
```

## Command-line backup

```bash
./scripts/backup.sh
# or
DATA_DIR=/path/to/data ./scripts/backup.sh
```
