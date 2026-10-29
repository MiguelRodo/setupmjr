package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/netutil"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

func getLatestGitHubReleaseTag(ownerRepo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", ownerRepo)
	resp, err := netutil.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return release.TagName, nil
}

func extractDevcontainer(archive io.Reader, repoDirName string) ([]byte, error) {
	cmd := exec.Command("tar", "xz", "--strip-components=1", "-C", ".", "--", repoDirName+"/.devcontainer")
	cmd.Stdin = archive
	return cmd.CombinedOutput()
}

func SetupRepoDevcontainer(repo, branch string, build bool) error {
	if repo == "" {
		repo = "MiguelRodo/comp"
	}
	if branch == "" {
		branch = "main"
	}

	tarURL := fmt.Sprintf("https://github.com/%s/archive/refs/heads/%s.tar.gz", repo, branch)
	repoNameParts := strings.Split(repo, "/")
	if len(repoNameParts) != 2 {
		return fmt.Errorf("invalid repo format, expected owner/name")
	}
	repoDirName := repoNameParts[1] + "-" + branch

	tmpDir, err := os.MkdirTemp("", "setupmjr-devcontainer-*")
	if err != nil {
		return fmt.Errorf("create temporary devcontainer download directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	tarPath := filepath.Join(tmpDir, "devcontainer.tar.gz")
	if err := netutil.DownloadFile(tarURL, tarPath); err != nil {
		return fmt.Errorf("failed to download .devcontainer from %s: %w", repo, err)
	}
	archive, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("open downloaded .devcontainer archive: %w", err)
	}
	defer archive.Close()

	output, err := extractDevcontainer(archive, repoDirName)
	if err != nil {
		return fmt.Errorf("failed to extract .devcontainer from %s: %v, output: %s", repo, err, string(output))
	}

	if build {
		if err := sysutil.EnsureDirExists(".github/workflows", 0755); err != nil {
			return err
		}
		tag, err := getLatestGitHubReleaseTag("MiguelRodo/actions")
		if err != nil {
			return fmt.Errorf("failed to fetch latest actions release: %w", err)
		}
		url := fmt.Sprintf("https://raw.githubusercontent.com/MiguelRodo/actions/%s/examples/prebuild-devcontainer.yml", tag)
		dest := filepath.Join(".github", "workflows", "prebuild-devcontainer.yml")
		if err := netutil.DownloadFile(url, dest); err != nil {
			return fmt.Errorf("failed to download prebuild-devcontainer.yml: %w", err)
		}
		fmt.Println("Successfully downloaded prebuild-devcontainer.yml")
	}

	fmt.Println("Successfully copied .devcontainer")
	return nil
}

func SetupRepoAction(actionName string) error {
	if err := sysutil.EnsureDirExists(".github/workflows", 0755); err != nil {
		return err
	}

	tag, err := getLatestGitHubReleaseTag("MiguelRodo/actions")
	if err != nil {
		return fmt.Errorf("failed to fetch latest actions release: %w", err)
	}
	url := fmt.Sprintf("https://raw.githubusercontent.com/MiguelRodo/actions/%s/examples/%s.yml", tag, actionName)
	dest := filepath.Join(".github", "workflows", actionName+".yml")
	if err := netutil.DownloadFile(url, dest); err != nil {
		return fmt.Errorf("failed to download action %s from MiguelRodo/actions@%s: %w", actionName, tag, err)
	}

	fmt.Printf("Successfully added %s from MiguelRodo/actions@%s to .github/workflows\n", actionName, tag)
	return nil
}

func SetupRepoInstallRepos() error {
	tag, err := getLatestGitHubReleaseTag("MiguelRodo/repos")
	if err != nil {
		return fmt.Errorf("failed to fetch latest repos release: %w", err)
	}
	version := strings.TrimPrefix(tag, "v")

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var ext string
	if goos == "windows" {
		ext = "zip"
	} else if goos == "linux" {
		ext = "tar.gz"
	} else {
		ext = "tar.gz"
	}

	assetName := fmt.Sprintf("repos_%s_%s_%s.%s", version, goos, goarch, ext)
	url := fmt.Sprintf("https://github.com/MiguelRodo/repos/releases/download/v%s/%s", version, assetName)

	tmpDir, err := os.MkdirTemp("", "setupmjr-repos-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, assetName)
	if err := netutil.DownloadFile(url, tmpFile); err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}

	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	binDir := filepath.Join(home, ".local", "bin")
	if err := sysutil.EnsureDirExists(binDir, 0755); err != nil {
		return err
	}

	destPath := filepath.Join(binDir, "repos")
	if goos == "windows" {
		destPath += ".exe"
	}

	if ext == "tar.gz" {
		cmd := exec.Command("tar", "xzf", tmpFile, "-C", tmpDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tar extraction failed: %w", err)
		}

		extractedBin := filepath.Join(tmpDir, "repos")
		if err := os.Rename(extractedBin, destPath); err != nil {
			return fmt.Errorf("failed to move binary to %s: %w", destPath, err)
		}
	} else if ext == "zip" {
		cmd := exec.Command("unzip", "-o", tmpFile, "-d", tmpDir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("unzip failed: %w", err)
		}
		extractedBin := filepath.Join(tmpDir, "repos.exe")
		if err := os.Rename(extractedBin, destPath); err != nil {
			return fmt.Errorf("failed to move binary to %s: %w", destPath, err)
		}
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make repos executable: %w", err)
	}

	fmt.Printf("Successfully installed repos to %s\n", destPath)
	return nil
}
