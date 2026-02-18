package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/mj1618/swarm-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	upgradeForce bool
	upgradeCheck bool
)

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Draft      bool          `json:"draft"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int    `json:"size"`
}

const githubRepo = "mj1618/swarm-cli"

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade swarm to the latest version",
	Long:  `Check for and install the latest version of swarm-cli from GitHub releases.`,
	Example: `  # Upgrade to latest version
  swarm upgrade

  # Check for updates without installing
  swarm upgrade --check

  # Force reinstall even if already on latest
  swarm upgrade --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgrade()
	},
}

func runUpgrade() error {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	currentVersion := version.Version
	bold.Printf("Current version: %s\n", currentVersion)

	fmt.Println("Checking for updates...")
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	bold.Printf("Latest version:  %s\n", latestVersion)
	fmt.Println()

	if !upgradeForce && currentVersion == latestVersion {
		green.Println("Already up to date!")
		return nil
	}

	if upgradeCheck {
		if currentVersion != latestVersion {
			fmt.Printf("Update available: %s → %s\n", currentVersion, latestVersion)
			fmt.Println("Run 'swarm upgrade' to install")
		}
		return nil
	}

	assetName := upgradeAssetName()
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s (looked for %s)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	fmt.Printf("Downloading %s...\n", assetName)

	tmpDir, err := os.MkdirTemp("", "swarm-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(archivePath, downloadURL); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	binaryName := "swarm"
	if runtime.GOOS == "windows" {
		binaryName = "swarm.exe"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)

	if strings.HasSuffix(assetName, ".zip") {
		err = extractZipBinary(archivePath, binaryPath, binaryName)
	} else {
		err = extractTarGzBinary(archivePath, binaryPath, binaryName)
	}
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find current executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	fmt.Printf("Installing to %s...\n", execPath)
	if err := replaceBinary(execPath, binaryPath); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing to %s (try running with sudo)", execPath)
		}
		return fmt.Errorf("failed to install: %w", err)
	}

	fmt.Println()
	green.Printf("Successfully upgraded to %s!\n", latestVersion)
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}

	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		if r.TagName == "latest" {
			continue
		}
		if !strings.HasPrefix(r.TagName, "v") {
			continue
		}
		return &r, nil
	}

	return nil, fmt.Errorf("no releases found")
}

func upgradeAssetName() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("swarm-cli_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func extractTarGzBinary(archivePath, destPath, binaryName string) error {
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
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if filepath.Base(header.Name) == binaryName && !header.FileInfo().IsDir() {
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
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
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractZipBinary(archivePath, destPath, binaryName string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, rc)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			return closeErr
		}
	}

	return fmt.Errorf("binary %q not found in archive", binaryName)
}

func replaceBinary(destPath, srcPath string) error {
	info, err := os.Stat(destPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	mode := os.FileMode(0755)
	if err == nil {
		mode = info.Mode()
	}

	tmpPath := destPath + ".new"

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmpPath)
		return err
	}
	dst.Close()

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeForce, "force", false, "Force reinstall even if already on the latest version")
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, "Only check for updates without installing")
}
