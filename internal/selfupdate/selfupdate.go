// Package selfupdate checks GitHub for a newer ntzh release and replaces the
// running binary in place. It speaks only to the public GitHub API and release
// download endpoints — no auth, stdlib only.
package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	repo      = "Nortezh/cli"
	latestURL = "https://api.github.com/repos/" + repo + "/releases/latest"
)

// Release is the subset of the GitHub release payload the CLI needs.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// LatestRelease fetches the latest published (non-draft) release from GitHub.
func LatestRelease(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("query github: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024))
		return nil, fmt.Errorf("github returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("github release has no tag")
	}
	return &rel, nil
}

// Newer reports whether remote is a strictly newer semver than current. A
// current version that is not parseable semver (e.g. "dev") is always treated
// as older, so local/dev builds are offered the update.
func Newer(current, remote string) bool {
	rc, rok := parseSemver(current)
	rr, rok2 := parseSemver(remote)
	if !rok2 {
		return false
	}
	if !rok {
		return true
	}
	for i := range 3 {
		if rr[i] != rc[i] {
			return rr[i] > rc[i]
		}
	}
	return false
}

// Apply downloads the release archive matching the running OS/arch, extracts
// the ntzh binary, and atomically replaces the running executable.
func Apply(ctx context.Context, tag string) error {
	name, isZip := archiveName(tag, runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s returned %d", name, resp.StatusCode)
	}

	archive, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}

	binName := "ntzh"
	if runtime.GOOS == "windows" {
		binName = "ntzh.exe"
	}
	var bin []byte
	if isZip {
		bin, err = extractZip(archive, binName)
	} else {
		bin, err = extractTarGz(archive, binName)
	}
	if err != nil {
		return err
	}

	return replaceExecutable(bin)
}

// archiveName builds the release asset name for a target, matching the
// name_template in .goreleaser.yaml (title-cased OS, x86_64 for amd64).
func archiveName(tag, goos, goarch string) (name string, isZip bool) {
	ver := strings.TrimPrefix(tag, "v")

	osName := goos
	if osName != "" {
		osName = strings.ToUpper(osName[:1]) + osName[1:]
	}

	arch := goarch
	if arch == "amd64" {
		arch = "x86_64"
	}

	if goos == "windows" {
		return fmt.Sprintf("ntzh_%s_%s_%s.zip", ver, osName, arch), true
	}
	return fmt.Sprintf("ntzh_%s_%s_%s.tar.gz", ver, osName, arch), false
}

func extractTarGz(archive []byte, binName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if filepath.Base(hdr.Name) == binName {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binName)
}

func extractZip(archive []byte, binName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) == binName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("%s not found in archive", binName)
}

// replaceExecutable writes bin next to the running executable and renames it
// over the current binary. On Unix the running process keeps its open inode, so
// the rename is safe; on Windows the old binary is moved aside first.
func replaceExecutable(bin []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)

	tmp, err := os.CreateTemp(dir, "ntzh-*.new")
	if err != nil {
		return fmt.Errorf("write to %s: %w (try re-running with sudo or use the install script)", dir, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		old := exe + ".old"
		_ = os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, exe); err != nil {
			_ = os.Rename(old, exe) // roll back
			return err
		}
		_ = os.Remove(old)
		return nil
	}

	if err := os.Rename(tmpPath, exe); err != nil {
		return fmt.Errorf("replace %s: %w (try re-running with sudo)", exe, err)
	}
	return nil
}

// parseSemver parses "1.2.3" or "v1.2.3" (ignoring any -prerelease/+build
// suffix) into [major, minor, patch].
func parseSemver(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
