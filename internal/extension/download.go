package extension

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const githubOrg = "braidsdev"

// binaryName returns the platform-specific binary name for an extension.
func binaryName(protocol string) string {
	return fmt.Sprintf("braids-ext-%s_%s_%s", protocol, runtime.GOOS, runtime.GOARCH)
}

// releaseBaseURL returns the GitHub releases latest download URL for a protocol.
func releaseBaseURL(protocol string) string {
	return fmt.Sprintf("https://github.com/%s/braids-ext-%s/releases/latest/download/", githubOrg, protocol)
}

// releaseLatestURL returns the GitHub releases /latest redirect URL for version checking.
func releaseLatestURL(protocol string) string {
	return fmt.Sprintf("https://github.com/%s/braids-ext-%s/releases/latest", githubOrg, protocol)
}

// Download fetches the extension binary from GitHub releases with checksum verification.
// If force is true, re-downloads even if the binary already exists.
func Download(protocol string, force bool) error {
	bp, err := binPath(protocol)
	if err != nil {
		return err
	}

	if !force {
		if _, err := os.Stat(bp); err == nil {
			return nil // already exists
		}
	}

	base := releaseBaseURL(protocol)
	name := binaryName(protocol)

	// Download checksums.txt
	expectedHash, err := fetchChecksum(base+"checksums.txt", name)
	if err != nil {
		return err
	}

	// Download binary to temp file
	binaryURL := base + name
	resp, err := http.Get(binaryURL)
	if err != nil {
		return fmt.Errorf("downloading extension %q: %w", protocol, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("extension %q is not yet available for download.\n"+
			"Extension binaries are published at github.com/%s/braids-ext-%s/releases.\n"+
			"Check back soon or build the extension from source.",
			protocol, githubOrg, protocol)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading extension %q: HTTP %d", protocol, resp.StatusCode)
	}

	dir, err := extensionsDir()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "download-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // clean up on error

	hasher := sha256.New()
	writer := io.MultiWriter(tmp, hasher)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmp.Close()
		return fmt.Errorf("writing extension binary: %w", err)
	}
	tmp.Close()

	// Verify checksum
	actualHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %q: expected %s, got %s", protocol, expectedHash, actualHash)
	}

	// chmod +x and atomic rename
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("setting executable permission: %w", err)
	}
	if err := os.Rename(tmpPath, bp); err != nil {
		return fmt.Errorf("installing extension binary: %w", err)
	}

	// Update manifest
	version, _ := CheckLatestVersion(protocol)
	if version == "" {
		version = "unknown"
	}
	return saveManifestEntry(protocol, ManifestEntry{
		Version:     version,
		InstalledAt: time.Now().UTC(),
		SHA256:      actualHash,
	})
}

// fetchChecksum downloads checksums.txt and returns the SHA256 for the named file.
func fetchChecksum(url, filename string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("downloading checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("checksums not found — extension may not be published yet")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading checksums: HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "<hash>  <filename>" or "<hash> <filename>"
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %q in checksums.txt", filename)
}

// CheckLatestVersion resolves the latest release tag via GitHub's /latest redirect.
// Returns the version string (e.g. "0.2.0") without "v" prefix, or error.
func CheckLatestVersion(protocol string) (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirects
		},
		Timeout: 5 * time.Second,
	}

	resp, err := client.Head(releaseLatestURL(protocol))
	if err != nil {
		return "", fmt.Errorf("checking latest version for %q: %w", protocol, err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("no releases found for extension %q", protocol)
	}

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("no redirect for extension %q releases", protocol)
	}

	// Location: https://github.com/braidsdev/braids-ext-postgres/releases/tag/v0.2.0
	idx := strings.LastIndex(loc, "/")
	if idx < 0 {
		return "", fmt.Errorf("unexpected redirect URL: %s", loc)
	}
	tag := loc[idx+1:]
	return strings.TrimPrefix(tag, "v"), nil
}
