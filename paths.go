package main

import (
	"os"
	"path/filepath"
	"strings"
)

// resolveDataRoot returns the portable data directory for Naraberu.
// Data lives next to the executable (or NARABERU_DATA_DIR), not in a machine-global config folder.
func resolveDataRoot() string {
	if env := strings.TrimSpace(os.Getenv("NARABERU_DATA_DIR")); env != "" {
		return filepath.Clean(env)
	}
	if exeDir, ok := executableDir(); ok && !isTempBuildDir(exeDir) {
		candidate := filepath.Join(exeDir, "data")
		if dirWritable(candidate) {
			return candidate
		}
	}
	return filepath.Join(".", "data")
}

// dirWritable reports whether the directory can be created and used for data.
// Falls back when the executable sits in a write-protected location.
func dirWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	probe := filepath.Join(dir, ".write-probe")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(probe)
	return true
}

func executableDir() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(exe), true
}

// isTempBuildDir reports whether dir looks like a go-build / temp location,
// which is common when running via `go run`.
func isTempBuildDir(dir string) bool {
	clean := filepath.Clean(dir)
	lower := strings.ToLower(clean)
	if strings.Contains(lower, string(filepath.Separator)+"go-build") {
		return true
	}
	if tmp, err := filepath.Abs(os.TempDir()); err == nil {
		abs, err := filepath.Abs(clean)
		if err == nil {
			tmp = filepath.Clean(tmp)
			if abs == tmp || strings.HasPrefix(abs, tmp+string(filepath.Separator)) {
				return true
			}
		}
	}
	return false
}

func dataSubdir(dataRoot, name string) string {
	return filepath.Join(dataRoot, name)
}

func dbPathIn(dataRoot string) string {
	return filepath.Join(dataRoot, "naraberu.db")
}

func thumbnailsDirIn(dataRoot string) string {
	return dataSubdir(dataRoot, "thumbnails")
}

func backupsDirIn(dataRoot string) string {
	return dataSubdir(dataRoot, "backups")
}

func soundsDirIn(dataRoot string) string {
	return dataSubdir(dataRoot, "sounds")
}

func webviewDirIn(dataRoot string) string {
	return dataSubdir(dataRoot, "webview")
}

func ensureDataLayout(dataRoot string) error {
	for _, name := range []string{"thumbnails", "backups", "sounds", "webview"} {
		if err := os.MkdirAll(dataSubdir(dataRoot, name), 0755); err != nil {
			return err
		}
	}
	return nil
}

// pathInsideDir reports whether candidate is dir or lives under dir.
func pathInsideDir(dir, candidate string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	absCand, err := filepath.Abs(candidate)
	if err != nil {
		return false
	}
	absDir = filepath.Clean(absDir)
	absCand = filepath.Clean(absCand)
	if absCand == absDir {
		return true
	}
	prefix := absDir + string(filepath.Separator)
	return strings.HasPrefix(absCand, prefix)
}
