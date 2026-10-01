# Release checklist (Windows, GitHub Releases)

Goal: a visitor downloads one zip from Releases, unpacks it in a folder he can write to, and runs `naraberu.exe` without extra setup beyond WebView2.

**Release 1.0.0** is the current target. Builds are **not code-signed**.

---

## 1. Before tagging

- [ ] `go test ./…` and `go vet ./…` pass
- [ ] `./scripts/package-release.sh` produces `dist/Naraberu-<version>-windows-amd64.zip`
- [ ] Zip contains `naraberu.exe`, `LICENSE`, `README.md`, `docs/`
- [ ] Smoke: unpack zip to a **writable** folder (not Program Files), run exe, confirm `data/` appears beside it and a series can be added
- [ ] README build/run/WebView2 notes still match the shipped binary

---

## 2. Version resources (exe Properties)

**State today:** `scripts/build-production.sh` writes `build/windows/info.generated.json` with product name, `Copyright Naraberu contributors`, and `VERSION` (default `1.0.0`), then runs `wails3 generate syso`. Whether those strings land in the PE Properties dialog can depend on the Go/Wails toolchain; **check after every release build**:

```powershell
(Get-Item build\bin\naraberu.exe).VersionInfo | Format-List *
```

Want: `ProductName` = Naraberu, `LegalCopyright` = Copyright Naraberu contributors, `FileVersion` / `ProductVersion` = the release version.

**If he sees blank properties (known gap on some toolchains):**

1. Confirm `build/windows/info.generated.json` has the right values (the build script writes this).
2. Rebuild with a clean `wails_windows.syso` (or `main_windows_amd64.syso`) in the package root while `go build` runs—do not delete it before link.
3. If Wails `generate syso` still does not link: use a standalone resource tool such as `goversioninfo` or `windres` to emit a `*.syso` for the main package, then `go build` again. Keep that inside `build-production.sh` only if needed; do not invent a second version story.

**Release versioning:**

- `VERSION=x.y.z` is the single knob (`package-release.sh` passes it through).
- Tag the repo `v<version>` (for example `v1.0.0`).
- Keep the GitHub release title and tag identical to `VERSION`.

---

## 3. Code signing

**This project ships unsigned** for 1.0. That is accepted: SmartScreen may show “Windows protected your PC”; the README “Getting started” section tells the user to choose **More info → Run anyway**.

Do not add a signing step to `package-release.sh` unless the project later buys a certificate. Keep release notes mentioning the warning.

---

## 4. GitHub release notes (template)

```markdown
## Naraberu vX.Y.Z

Offline desktop anime tracker. Unpack the zip into a folder you can write to, then run `naraberu.exe`.

### Windows requirements
- Windows 10 or 11 (x64)
- [WebView2 runtime](https://developer.microsoft.com/microsoft-edge/webview2/) (preinstalled on most up-to-date systems)

### Where data lives
Naraberu stores everything in a `data` folder **next to** `naraberu.exe` (SQLite database, thumbnails, backups, sounds). To put data elsewhere, set `NARABERU_DATA_DIR` before starting the app.

Do not install the exe under Program Files unless you also set `NARABERU_DATA_DIR`; the app needs a writable data folder.

### Notes
- SmartScreen may warn on unsigned builds (unless this release is signed). Choose More info → Run anyway if you trust the source.
- License: AGPL-3.0 (see LICENSE in the zip).

### Downloads
| File | Description |
|------|-------------|
| `Naraberu-X.Y.Z-windows-amd64.zip` | App, license, readme, docs |
```

Adjust the table name to the zip `package-release.sh` produced.

---

## 5. Publish steps

1. [ ] Create the tag `vX.Y.Z` (or draft a release from a tag).
2. [ ] Upload `dist/Naraberu-X.Y.Z-windows-amd64.zip`.
3. [ ] Paste release notes (section 4).
4. [ ] Mark as latest release if appropriate.
5. [ ] Download the zip into a clean folder and run once as a user would.

---

## 6. Out of scope for a first release

- NSIS installer (`build/windows/installer` stubs remain unused unless you opt in)
- Code signing if no cert exists yet
- Automatic GitHub Actions (can call `package-release.sh` later)
- macOS / Linux builds (Wails templates exist; not packaged here)
