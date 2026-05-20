package bash

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/assets"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

// SetupBashRCD ensures ~/.bashrc.d is sourced by adding sourcing block to ~/.bashrc.
func SetupBashRCD() error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	// 1. Add sourcing block to ~/.bashrc
	bashrcPath := filepath.Join(home, ".bashrc")
	sourcingBlock := `for i in $HOME/.bashrc.d/*; do [ -r "$i" ] && source "$i"; done`

	b, err := os.ReadFile(bashrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .bashrc: %w", err)
	}

	hasBashrcD := false
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, ".bashrc.d") {
			hasBashrcD = true
			break
		}
	}

	if !hasBashrcD {
		f, err := os.OpenFile(bashrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open .bashrc: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString("\n" + sourcingBlock + "\n"); err != nil {
			return fmt.Errorf("write to .bashrc: %w", err)
		}
		fmt.Println("Added .bashrc.d sourcing block to ~/.bashrc")
	} else {
		fmt.Println("~/.bashrc already sources .bashrc.d")
	}

	// 2. Ensure ~/.bashrc.d exists
	bashrcd := filepath.Join(home, ".bashrc.d")
	if err := sysutil.EnsureDirExists(bashrcd, 0755); err != nil {
		return err
	}

	return nil
}

// SetupBashPath copies path-executables.sh to ~/.bashrc.d.
func SetupBashPath() error {
	if err := SetupBashRCD(); err != nil {
		return err
	}

	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	bashrcd := filepath.Join(home, ".bashrc.d")
	dst := filepath.Join(bashrcd, "path-executables.sh")
	if err := sysutil.CopyEmbedFile(assets.FS, "bashrc.d/path-executables.sh", dst, 0755); err != nil {
		return fmt.Errorf("copy path-executables.sh: %w", err)
	}
	fmt.Printf("Copied path-executables.sh to %s\n", dst)

	return nil
}

// SetupBashLogin copies login.sh to ~/.bashrc.d.
func SetupBashLogin() error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	bashrcd := filepath.Join(home, ".bashrc.d")
	if err := sysutil.EnsureDirExists(bashrcd, 0755); err != nil {
		return err
	}

	dst := filepath.Join(bashrcd, "login.sh")
	// Check if login.sh already exists, if so skip overwriting it
	if _, err := os.Stat(dst); err == nil {
		fmt.Printf("login.sh already exists at %s, skipping copy\n", dst)
		return nil
	}

	if err := sysutil.CopyEmbedFile(assets.FS, "bashrc.d/login.sh", dst, 0755); err != nil {
		return fmt.Errorf("copy login.sh: %w", err)
	}
	fmt.Printf("Copied login.sh to %s\n", dst)

	return nil
}
