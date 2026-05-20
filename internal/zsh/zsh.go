package zsh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/assets"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

// SetupZshRCD ensures ~/.zshrc.d is sourced by ~/.zshrc,
// and copies path-executables.sh to ~/.zshrc.d.
func SetupZshRCD() error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	// 1. Add sourcing block to ~/.zshrc
	zshrcPath := filepath.Join(home, ".zshrc")
	sourcingBlock := `for i in $HOME/.zshrc.d/*; do [ -r "$i" ] && source "$i"; done`

	b, err := os.ReadFile(zshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .zshrc: %w", err)
	}

	hasZshrcD := false
	for _, line := range strings.Split(string(b), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, ".zshrc.d") {
			hasZshrcD = true
			break
		}
	}

	if !hasZshrcD {
		f, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open .zshrc: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString("\n" + sourcingBlock + "\n"); err != nil {
			return fmt.Errorf("write to .zshrc: %w", err)
		}
		fmt.Println("Added .zshrc.d sourcing block to ~/.zshrc")
	} else {
		fmt.Println("~/.zshrc already sources .zshrc.d")
	}

	// 2. Ensure ~/.zshrc.d exists
	zshrcd := filepath.Join(home, ".zshrc.d")
	if err := sysutil.EnsureDirExists(zshrcd, 0755); err != nil {
		return err
	}

	// 3. Copy path-executables.sh
	dst := filepath.Join(zshrcd, "path-executables.sh")
	if err := sysutil.CopyEmbedFile(assets.FS, "bashrc.d/path-executables.sh", dst, 0755); err != nil {
		return fmt.Errorf("copy path-executables.sh: %w", err)
	}
	fmt.Printf("Copied path-executables.sh to %s\n", dst)

	return nil
}

// SetupZshLogin copies login.sh to ~/.zshrc.d.
func SetupZshLogin() error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	zshrcd := filepath.Join(home, ".zshrc.d")
	if err := sysutil.EnsureDirExists(zshrcd, 0755); err != nil {
		return err
	}

	dst := filepath.Join(zshrcd, "login.sh")
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
