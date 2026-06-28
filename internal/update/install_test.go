package update_test

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"isobox/internal/update"
)

func TestPrepareReleaseVerifiesChecksumAndSmokeTestsExtractedBinary(t *testing.T) {
	dir := t.TempDir()
	archive := makeReleaseArchive(t, dir, "v1.2.3")
	checksum := writeChecksums(t, dir, archive)

	prepared, err := update.PrepareRelease(update.Release{TagName: "v1.2.3", Assets: []update.Asset{
		{Name: filepath.Base(archive), URL: "file://" + archive},
		{Name: "checksums.txt", URL: "file://" + checksum},
	}}, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatalf("PrepareRelease: %v", err)
	}
	defer prepared.Cleanup()
	out, err := update.RunVersion(prepared.BinaryPath)
	if err != nil || strings.TrimSpace(out) != "v1.2.3" {
		t.Fatalf("prepared binary version = %q, %v", out, err)
	}
}

func TestPrepareReleaseChecksumMismatchAbortsBeforeExtraction(t *testing.T) {
	dir := t.TempDir()
	archive := makeReleaseArchive(t, dir, "v1.2.3")
	checksum := filepath.Join(dir, "checksums.txt")
	if err := os.WriteFile(checksum, []byte("0000000000000000000000000000000000000000000000000000000000000000  "+filepath.Base(archive)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := update.PrepareRelease(update.Release{TagName: "v1.2.3", Assets: []update.Asset{{Name: filepath.Base(archive), URL: "file://" + archive}, {Name: "checksums.txt", URL: "file://" + checksum}}}, runtime.GOOS, runtime.GOARCH)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v, want checksum mismatch", err)
	}
}

func TestPrepareReleaseVersionMismatchAborts(t *testing.T) {
	dir := t.TempDir()
	archive := makeReleaseArchive(t, dir, "v9.9.9")
	checksum := writeChecksums(t, dir, archive)
	_, err := update.PrepareRelease(update.Release{TagName: "v1.2.3", Assets: []update.Asset{{Name: filepath.Base(archive), URL: "file://" + archive}, {Name: "checksums.txt", URL: "file://" + checksum}}}, runtime.GOOS, runtime.GOARCH)
	if err == nil || !strings.Contains(err.Error(), "reports version") {
		t.Fatalf("err = %v, want version mismatch", err)
	}
}

func TestInstallPreparedReleaseBacksUpReplacesSmokeTestsAndCleansBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "isobox")
	writeVersionScript(t, target, "v1.0.0")
	prepared := filepath.Join(dir, "new-isobox")
	writeVersionScript(t, prepared, "v1.2.3")
	result, err := update.InstallPreparedRelease(update.PreparedRelease{BinaryPath: prepared, Version: "v1.2.3"}, update.Target{Path: target})
	if err != nil {
		t.Fatalf("InstallPreparedRelease: %v", err)
	}
	if result.InstalledVersion != "v1.2.3" {
		t.Fatalf("installed = %q", result.InstalledVersion)
	}
	if _, err := os.Stat(result.BackupPath); !os.IsNotExist(err) {
		t.Fatalf("backup still exists or stat err = %v", err)
	}
	out, _ := update.RunVersion(target)
	if strings.TrimSpace(out) != "v1.2.3" {
		t.Fatalf("target version = %q", out)
	}
}

func TestInstallPreparedReleaseRollsBackWhenPostReplacementSmokeTestFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "isobox")
	writeVersionScript(t, target, "v1.0.0")
	bad := filepath.Join(dir, "bad-isobox")
	if err := os.WriteFile(bad, []byte("#!/bin/sh\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := update.InstallPreparedRelease(update.PreparedRelease{BinaryPath: bad, Version: "v1.2.3"}, update.Target{Path: target})
	if err == nil || !strings.Contains(err.Error(), "rolled back") {
		t.Fatalf("err = %v, want rollback", err)
	}
	out, _ := update.RunVersion(target)
	if strings.TrimSpace(out) != "v1.0.0" {
		t.Fatalf("target was not rolled back, version %q", out)
	}
}

func makeReleaseArchive(t *testing.T, dir, version string) string {
	t.Helper()
	bin := filepath.Join(dir, "isobox")
	writeVersionScript(t, bin, version)
	archive := filepath.Join(dir, "isobox_"+runtime.GOOS+"_"+runtime.GOARCH+".tar.gz")
	f, _ := os.Create(archive)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	info, _ := os.Stat(bin)
	hdr, _ := tar.FileInfoHeader(info, "")
	hdr.Name = "isobox"
	tw.WriteHeader(hdr)
	b, _ := os.ReadFile(bin)
	tw.Write(b)
	tw.Close()
	gz.Close()
	f.Close()
	return archive
}
func writeChecksums(t *testing.T, dir, archive string) string {
	t.Helper()
	b, _ := os.ReadFile(archive)
	sum := sha256.Sum256(b)
	path := filepath.Join(dir, "checksums.txt")
	os.WriteFile(path, []byte(fmt.Sprintf("%x  %s\n", sum, filepath.Base(archive))), 0o644)
	return path
}
func writeVersionScript(t *testing.T, path, version string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nif [ \"$1\" = version ]; then echo "+version+"; exit 0; fi\nexit 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
