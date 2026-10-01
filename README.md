# Naraberu

**Version 1.0.0**—offline desktop anime tracker (Windows x64).

Naraberu is a desktop app for keeping a personal anime list on your own machine. Add series, mark what you have watched, score them, write notes, tag them, and find them again later—without an account, without a server, and without anyone else reading your list.

It is built for people who want a calm, local library: open the app, see what is there, change what you want, close it. Everything lives in one folder you can copy to a USB drive or a backup disk.

## At a glance

| | |
|---|---|
| Interface | Desktop window (Wails and a small HTML/CSS/JS front end) |
| Storage | One SQLite file under a portable `data` folder |
| Direct Go dependencies | 4 (Wails v3, uuid, modernc sqlite, x/text) |
| JavaScript / CSS frameworks | None |
| Color themes | 10 |
| Automated tests | 13 |
| Approx. source size | ~4,000 lines of Go and front-end code |
| Typical Windows build | ~16 MB |

## Features

- **Series**: English name, Japanese name, alternative names
- **Status**: To Watch, In Progress, Watched, Abandoned
- **Score**: one number from 0 to 10 (see Data fields below)
- **Favorites** and **Owned** toggles
- **Links**: connect sequels and spin-offs; links go both ways
- **Thumbnails**: pick a local image, or let CSV import fetch a URL
- **Dates**: start and end of watching, with flexible precision
- **Review and synopsis**: plain text or Markdown (headings, lists, bold, italics, monospace)
- **Tags**: free-form, de-duplicated, sorted
- **Search**: accent- and quote-insensitive; covers names, tags, synopsis, and notes
- **Sort and filters**: status, tags, favorites, owned, linked; English sort ignores leading “The”, “A”, “An”
- **CSV export and import**: with duplicate detection
- **Batch tools**: multi-select, batch delete, batch add tags
- **Themes and click sound**: ten themes; optional mp3 on list clicks
- **Keyboard**: see shortcuts below

## What Naraberu is not

Naraberu is **local, offline, and private** by design. It will not:

- Sign in to MyAnimeList, AniList, Kitsu, or any other tracker
- Sync with streaming services or record playback automatically
- Upload your list, scores, or notes anywhere
- Require an internet connection for normal use

That is intentional: Naraberu stays a private library on your machine.

If you want online sync or community features, other open-source projects fill that role: for example **Taiga**, **Trackma**, or **MAL-Sync**. They are aimed at multi-service tracking and live updates. Naraberu is intentionally the opposite: one file, one machine, no network unless you ask for a convenience.

### Optional network use

Two features touch the network. Neither is required for day-to-day use:

1. **CSV import of thumbnail URLs**: if a CSV row’s `thumbnailPath` is an `http(s)` URL, import may download that image into `data/thumbnails/`.
2. **`csvgen`**: a helper that looks up media on the web to build an import CSV.

If you do not use `csvgen` and you import only local image paths (no thumbnail URLs), Naraberu never needs the network at all.

## Data fields

| Field | Meaning |
|-------|---------|
| **English name / Japanese name** | Primary titles. Both are searchable. |
| **Alternative names** | Extra titles or romanizations; searchable. |
| **Status** | To Watch, In Progress, Watched, or Abandoned. |
| **Score** | A single score from **0 to 10**. You may use integers or a decimal (7, 7.5). Zero means “not scored”; the list shows a star badge only when the score is above zero. Out-of-range values on CSV import are reset to 0 and reported. |
| **Start date / End date** | When you watched it. Use the precision you have: `2024`, `2024/03`, or `2024/03/15`. Hyphens work the same (`2024-03-15`). The app does not guess missing parts. |
| **Review / notes** | Free text; rendered as Markdown when you are not editing. |
| **Synopsis** | Description; same Markdown behavior. |
| **Tags** | Labels such as `mecha` or `rewatch`. Matching is case-insensitive; the same tag is not listed twice. |
| **Favorite** | Heart toggle. |
| **Owned** | Star toggle, for series you have on Blu-ray, DVD, etc. |
| **Linked series** | Other entries (sequels, films, spin-offs). Editing a link updates both rows. |
| **Thumbnail** | Local file under `data/thumbnails/`, or a URL recorded until import downloads it. |
| **Created / last modified** | Shown at the bottom of the detail view. |

## Data storage and backups

By default, Naraberu uses a **`data` folder next to the executable**. It does not write to a machine-wide config directory.

| Path | Contents |
|------|----------|
| `data/naraberu.db` | SQLite database (series and settings) |
| `data/thumbnails/` | Image files |
| `data/backups/` | Automatic startup copies (keeps the last 5) |
| `data/sounds/` | Extra click sounds you add |
| `data/webview/` | Browser profile for the embedded view |

To put data somewhere else, set **`NARABERU_DATA_DIR`** before launching:

```bash
# Git Bash
NARABERU_DATA_DIR=/d/anime/naraberu-data ./naraberu.exe
```

If the folder beside the executable is not writable (for example under Program Files), Naraberu falls back to `./data` in the working directory. When you run via `go run`, data also lands under the project’s `./data` unless `NARABERU_DATA_DIR` is set.

Details: [docs/data-layout.md](docs/data-layout.md).

### Backups

The app copies `naraberu.db` into `data/backups/` on startup and keeps the five newest copies. That protects against a corrupt file or a bad edit.

**Recommendation:** now and then, use **Export CSV** and keep that file somewhere else (another disk, a folder you trust in the cloud, and so on). Automated backups sit beside the live database—they do not help if the whole folder is lost. A CSV export is a portable, human-readable safety copy.

You can also run:

```bash
./scripts/backup.sh
# or
DATA_DIR=/path/to/data ./scripts/backup.sh
```

## Click sound (audio)

List clicks can play a short mp3.

- The project ships a default clip as **`frontend/dist/pick.mp3`** (embedded at build time).
- To use your own sound: put an **`.mp3`** file in **`data/sounds/`**, open **Settings → Click Sound**, and choose it from the list. Files in that folder appear automatically.
- You can also pick **None** or tick **Mute**.

There is no volume slider yet (see Roadmap). Keep clips short; they play at a modest fixed level.

To replace the bundled default, overwrite `frontend/dist/pick.mp3` and rebuild so the new file is embedded.

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `/` | Focus search |
| `n` | Add series |
| `Escape` | Close a modal, or return to the list |
| `Backspace` | Clear search, status, favorites, linked, owned, and tags |
| Arrow Up / Down | Move between series cards |
| `Enter` | Open the focused card |
| `Tab` | Move between controls (modals keep focus inside) |

## Settings

The gear button opens settings for:

- **Color theme**: dark, light, and pink / red / green / blue in dark and light variants
- **Click sound**: bundled clip, any mp3 in `data/sounds/`, or None
- **Mute**: silence clicks without changing the selected file

Settings are stored in the database and apply to this install only.

## CSV

Export and import share one column list, including a single **`score`**. Quoted fields may contain commas and line breaks (useful for long reviews). See [docs/csv-format.md](docs/csv-format.md).

Import skips titles that already exist (by English name) and reports skips and warnings.

## csvgen (optional web helper)

`cmd/csvgen` turns a list of titles into a CSV ready for import. It is a convenience for bulk entry, not part of the main app.

### What it does

1. Reads a text file, **one series title per line** (sample: [docs/anime-list.example.txt](docs/anime-list.example.txt)).
2. Looks up each title (MyAnimeList via the Jikan API, with Wikipedia as fallback).
3. Writes a CSV with names, tags, synopsis, and thumbnail URLs when it can.

```bash
go run ./cmd/csvgen docs/anime-list.example.txt anime-import.csv
```

### How useful is it?

It is **decent at English and Japanese names** when the title is unambiguous. That is the main win for a long list.

Expect to **fix or replace thumbnails and synopses yourself** often. Automatic matching can pick the wrong show (short or odd titles are especially risky), and synopses are whatever the source returns—sometimes truncated or generic. Spot-check the CSV before import. Dates are left empty on purpose; those are your watch dates, not air dates.

The tool hits the network for every title and keeps a slow request pace. It does not compare against your existing database; import does that.

### Limits

- Fuzzy title matching will not get everything right
- No guarantee of a good image or a useful synopsis
- Re-running overwrites the output file
- Optional helper only; the app never requires it

## Getting started (download and run)

For users who just want the app (not the source):

1. **Download** `Naraberu-1.0.0-windows-amd64.zip` from the GitHub **Releases** page (or build it yourself—see below).
2. **Unpack** the zip into a folder you can write to, for example `D:\Apps\Naraberu` or on the Desktop. Avoid `Program Files` unless you also set `NARABERU_DATA_DIR`.
3. **If Windows says the app is unrecognized** (SmartScreen: “Windows protected your PC”):
   - Click **More info**
   - Click **Run anyway**
   - Builds are not code-signed; this warning is expected until a signed release exists.
4. **Run `naraberu.exe`.** On first start it creates a `data` folder next to the exe (database, thumbnails, backups).
5. **If the window does not open**, install the [WebView2 runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (usually already present on Windows 10/11), then start the app again.

No installer is required. To uninstall, delete the folder (and any separate folder you pointed `NARABERU_DATA_DIR` at).

## Requirements

**Using a release build:** Windows 10/11 x64 and WebView2 (see Getting started). Go is not required.

**Building from source:**

- Go 1.27 or newer
- [Wails v3](https://wails.io) CLI (`wails3`)
- Windows: [WebView2](https://developer.microsoft.com/microsoft-edge/webview2/) runtime

## Building and running

```bash
go run .
# or
wails3 task run
```

Everyday build (no embedded Windows icon resources):

```bash
go build -trimpath -ldflags "-H windowsgui" -o bin/naraberu.exe .
# or
wails3 task build
```

Production build (icon and version resources), from Git Bash:

```bash
./scripts/build-production.sh
# output: build/bin/naraberu.exe
```

Release zip (what you upload to GitHub Releases):

```bash
./scripts/package-release.sh
# or
VERSION=1.0.0 ./scripts/package-release.sh
# output: dist/Naraberu-<version>-windows-amd64.zip
```

Step-by-step publishing notes: [docs/release-checklist.md](docs/release-checklist.md).

This project targets Wails v3.

## Roadmap

Ideas for later development, not commitments:

- **Internationalization**: UI strings in more than English
- **Click-sound volume**: a real volume control, not only mute and unmute

## License

Copyright Naraberu contributors.

Naraberu is free software under the GNU Affero General Public License v3.0 (AGPL-3.0). See [LICENSE](LICENSE).
