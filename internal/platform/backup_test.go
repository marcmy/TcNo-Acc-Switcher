package platform

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShouldIncludeBackupFile_IncludeOverridesIgnore(t *testing.T) {
	include := normalizeExtensionSet([]string{".cfg", "json"})
	ignore := normalizeExtensionSet([]string{".cfg", ".log"})

	if !shouldIncludeBackupFile("a.cfg", include, ignore, false) {
		t.Fatal("expected .cfg to be included by include list")
	}
	if shouldIncludeBackupFile("a.log", include, ignore, false) {
		t.Fatal("expected .log to be excluded when not included")
	}
	if shouldIncludeBackupFile("a.txt", include, ignore, false) {
		t.Fatal("expected .txt to be excluded when include list is set")
	}
}

func TestShouldIncludeBackupFile_IgnoreOnly(t *testing.T) {
	ignore := normalizeExtensionSet([]string{".log"})
	if shouldIncludeBackupFile("keep.txt", nil, ignore, false) == false {
		t.Fatal("expected non-ignored extension to be kept")
	}
	if shouldIncludeBackupFile("skip.log", nil, ignore, false) {
		t.Fatal("expected ignored extension to be skipped")
	}
}

func TestSanitizeBackupRelativePathRejectsUnsafe(t *testing.T) {
	if _, err := sanitizeBackupRelativePath(`..\escape`, `C:\source`); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
	if _, err := sanitizeBackupRelativePath(`C:\abs\path`, `C:\source`); err == nil {
		t.Fatal("expected absolute path to be rejected")
	}
}

func TestSafeJoinZipPathRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	if _, err := safeJoinZipPath(base, `..\..\evil.txt`); err == nil {
		t.Fatal("expected traversal zip entry to be rejected")
	}
	if _, err := safeJoinZipPath(base, `C:\abs\path.txt`); err == nil {
		t.Fatal("expected absolute zip entry to be rejected")
	}
}

func TestLatestBackupZipSelectsNewest(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "Backup_EpicGames_2000-01-01_00-00-00.zip")
	newPath := filepath.Join(dir, "Backup_EpicGames_2001-01-01_00-00-00.zip")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	got, err := latestBackupZip(dir, "EpicGames")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(newPath) {
		t.Fatalf("expected newest backup, got %q", got)
	}
}

func TestExtractZipSafelyRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "restore")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dest, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	archive := filepath.Join(base, "backup.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("escape/evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("escaped")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := extractZipSafely(archive, dest); err == nil {
		t.Fatal("archive extraction followed a symlink outside the root")
	}
	if _, err := os.Stat(filepath.Join(outside, "evil.txt")); !os.IsNotExist(err) {
		t.Fatalf("archive wrote outside the root: %v", err)
	}
}
