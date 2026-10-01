# Build Directory

Build assets for Naraberu (Wails v3).

* `bin` — production output (`naraberu.exe`)
* `darwin` — macOS Info.plist templates
* `windows` — icon, version info, manifest, installer stubs

## Windows

`windows/` holds the manifest and version resources used by `./scripts/build-production.sh`.

- `icon.ico` — application icon (generated from `appicon.png` when missing)
- `info.json` — product name and version for the exe properties and installer
- `wails.exe.manifest` — application manifest
- `installer/*` — optional NSIS installer templates

## macOS

`darwin/Info.plist` and `Info.dev.plist` are templates for Mac builds with Wails v3.
