package security

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func archiveForUnpackTest(t *testing.T, name string) []byte {
	t.Helper()
	var result bytes.Buffer
	gz := gzip.NewWriter(&result)
	writer := tar.NewWriter(gz)
	payload := []byte("private account data")
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(payload)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return result.Bytes()
}

func TestUnpackDirRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "unpack")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{dest, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(dest, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := unpackDir(archiveForUnpackTest(t, "link/secret.txt"), dest); err == nil {
		t.Fatal("tar extraction followed a symlink outside the root")
	}
	if _, err := os.Stat(filepath.Join(outside, "secret.txt")); !os.IsNotExist(err) {
		t.Fatalf("tar extraction escaped the root: %v", err)
	}
}

func TestUnpackDirAcceptsNestedFile(t *testing.T) {
	dest := t.TempDir()
	if err := unpackDir(archiveForUnpackTest(t, "nested/secret.txt"), dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "nested", "secret.txt"))
	if err != nil || string(got) != "private account data" {
		t.Fatalf("nested archive round trip = %q, %v", got, err)
	}
}

func TestUnpackDirRejectsParentTraversal(t *testing.T) {
	dest := t.TempDir()
	if err := unpackDir(archiveForUnpackTest(t, "../escape.txt"), dest); err == nil {
		t.Fatal("tar extraction accepted a parent traversal")
	}
}
