package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PreparedRelease struct{ BinaryPath, Version, tempDir string }

func (p PreparedRelease) Cleanup() error {
	if p.tempDir == "" {
		return nil
	}
	return os.RemoveAll(p.tempDir)
}

type InstallResult struct{ InstalledVersion, BackupPath string }

func PrepareRelease(release Release, goos, goarch string) (PreparedRelease, error) {
	archiveName := fmt.Sprintf("isobox_%s_%s.tar.gz", goos, goarch)
	archiveAsset, ok := findAsset(release.Assets, archiveName)
	if !ok {
		return PreparedRelease{}, fmt.Errorf("release %s has no asset %s", release.TagName, archiveName)
	}
	checksumAsset, ok := findAsset(release.Assets, "checksums.txt")
	if !ok {
		return PreparedRelease{}, fmt.Errorf("release %s has no checksums.txt asset", release.TagName)
	}
	tmp, err := os.MkdirTemp("", "isobox-update-*")
	if err != nil {
		return PreparedRelease{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(tmp)
		}
	}()
	archivePath := filepath.Join(tmp, archiveName)
	if err := downloadAsset(archiveAsset.URL, archivePath); err != nil {
		return PreparedRelease{}, fmt.Errorf("download archive: %w", err)
	}
	checksumPath := filepath.Join(tmp, "checksums.txt")
	if err := downloadAsset(checksumAsset.URL, checksumPath); err != nil {
		return PreparedRelease{}, fmt.Errorf("download checksums: %w", err)
	}
	if err := verifyChecksum(archivePath, checksumPath); err != nil {
		return PreparedRelease{}, err
	}
	binPath := filepath.Join(tmp, "isobox")
	if err := extractBinary(archivePath, binPath); err != nil {
		return PreparedRelease{}, err
	}
	out, err := RunVersion(binPath)
	if err != nil {
		return PreparedRelease{}, fmt.Errorf("smoke-test extracted binary: %w", err)
	}
	if strings.TrimSpace(out) != release.TagName {
		return PreparedRelease{}, fmt.Errorf("extracted binary reports version %q, want %q", strings.TrimSpace(out), release.TagName)
	}
	cleanup = false
	return PreparedRelease{BinaryPath: binPath, Version: release.TagName, tempDir: tmp}, nil
}

func InstallPreparedRelease(prepared PreparedRelease, target Target) (InstallResult, error) {
	backup := target.Path + ".bak"
	_ = os.Remove(backup)
	if err := os.Rename(target.Path, backup); err != nil {
		return InstallResult{}, fmt.Errorf("backup update target: %w", err)
	}
	rollback := func(reason error) (InstallResult, error) {
		if err := os.Rename(backup, target.Path); err != nil {
			return InstallResult{BackupPath: backup}, fmt.Errorf("%v; rollback failed: %w", reason, err)
		}
		return InstallResult{BackupPath: backup}, fmt.Errorf("%v; rolled back to previous binary", reason)
	}
	mode := os.FileMode(0o755)
	if info, err := os.Stat(backup); err == nil {
		mode = info.Mode()&0o777 | 0o111
	}
	if err := copyFile(prepared.BinaryPath, target.Path, mode); err != nil {
		return rollback(fmt.Errorf("replace update target: %w", err))
	}
	out, err := RunVersion(target.Path)
	if err != nil {
		return rollback(fmt.Errorf("post-replacement smoke-test: %w", err))
	}
	got := strings.TrimSpace(out)
	if got != prepared.Version {
		return rollback(fmt.Errorf("post-replacement binary reports version %q, want %q", got, prepared.Version))
	}
	if err := os.Remove(backup); err != nil {
		return InstallResult{InstalledVersion: got, BackupPath: backup}, fmt.Errorf("delete backup: %w", err)
	}
	return InstallResult{InstalledVersion: got, BackupPath: backup}, nil
}

func RunVersion(path string) (string, error) {
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return string(out), err
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	line = strings.TrimPrefix(line, "isobox ")
	return line + "\n", nil
}
func findAsset(assets []Asset, name string) (Asset, bool) {
	for _, a := range assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}
func downloadAsset(rawURL, dest string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme == "file" {
		return copyFile(u.Path, dest, 0o644)
	}
	resp, err := http.Get(rawURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
func verifyChecksum(archivePath, checksumsPath string) error {
	b, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(b))
	lines, err := os.ReadFile(checksumsPath)
	if err != nil {
		return err
	}
	name := filepath.Base(archivePath)
	for _, line := range strings.Split(string(lines), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == name {
			if fields[0] != sum {
				return fmt.Errorf("checksum mismatch for %s", name)
			}
			return nil
		}
	}
	return fmt.Errorf("checksum for %s not found", name)
}
func extractBinary(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) != "isobox" || hdr.FileInfo().IsDir() {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode()|0o111)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return fmt.Errorf("archive does not contain isobox binary")
}
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}
