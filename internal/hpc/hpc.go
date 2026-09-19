package hpc

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/assets"
	"github.com/MiguelRodo/setupmjr/internal/gitcmd"
	"github.com/MiguelRodo/setupmjr/internal/r"
	"github.com/MiguelRodo/setupmjr/internal/shell"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

// SetupHPC runs the master HPC setup.
func SetupHPC() error {
	return setupHPC(runtime.GOOS)
}

func setupHPC(goos string) error {
	if goos != "linux" {
		return fmt.Errorf("HPC setup is Linux-only")
	}

	fmt.Println("Running master HPC setup...")

	if err := shell.SetupShellPath("bash"); err != nil {
		return err
	}
	if err := shell.SetupShellLogin("bash", false); err != nil {
		return err
	}
	if err := shell.SetupShellPath("zsh"); err != nil {
		return err
	}
	if err := shell.SetupShellLogin("zsh", false); err != nil {
		return err
	}
	if err := gitcmd.SetupGit(); err != nil {
		return err
	}
	if err := SetupHPCScratch(); err != nil {
		return err
	}
	if err := SetupHPCApptainer(); err != nil {
		return err
	}
	if err := SetupHPCSlurm(); err != nil {
		return err
	}
	return r.SetupR(false, false, false)
}

// SetupHPCScratch copies the hpc-scratch.sh script.
func SetupHPCScratch() error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	bashrcd := filepath.Join(home, ".bashrc.d")
	if err := sysutil.EnsureDirExists(bashrcd, 0755); err != nil {
		return err
	}

	dst := filepath.Join(bashrcd, "hpc-scratch.sh")
	if err := sysutil.CopyEmbedFile(assets.FS, "bashrc.d/hpc-scratch.sh", dst, 0755); err != nil {
		return fmt.Errorf("copy hpc-scratch.sh: %w", err)
	}
	fmt.Printf("Copied hpc-scratch.sh to %s\n", dst)

	return nil
}

// SetupHPCApptainer copies apptainer scripts to ~/.local/bin.
func SetupHPCApptainer() error {
	if err := shell.SetupShellPath("bash"); err != nil {
		return err
	}

	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	localBin := filepath.Join(home, ".local", "bin")
	if err := sysutil.EnsureDirExists(localBin, 0755); err != nil {
		return err
	}

	entries, err := fs.ReadDir(assets.FS, "scripts")
	if err != nil {
		return fmt.Errorf("read embedded scripts dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "apptainer-") && name != "apptainer-vscode" {
			src := path.Join("scripts", name)
			dst := filepath.Join(localBin, name)
			if err := sysutil.CopyEmbedFile(assets.FS, src, dst, 0755); err != nil {
				return fmt.Errorf("copy %s: %w", name, err)
			}
			fmt.Printf("Copied %s to %s\n", name, dst)
		}
	}

	return nil
}

// SetupHPCSlurm copies slurm scripts to ~/.local/bin.
func SetupHPCSlurm() error {
	if err := shell.SetupShellPath("bash"); err != nil {
		return err
	}

	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	localBin := filepath.Join(home, ".local", "bin")
	if err := sysutil.EnsureDirExists(localBin, 0755); err != nil {
		return err
	}

	entries, err := fs.ReadDir(assets.FS, "scripts")
	if err != nil {
		return fmt.Errorf("read embedded scripts dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "slurm-") {
			src := path.Join("scripts", name)
			dst := filepath.Join(localBin, name)
			if err := sysutil.CopyEmbedFile(assets.FS, src, dst, 0755); err != nil {
				return fmt.Errorf("copy %s: %w", name, err)
			}
			fmt.Printf("Copied %s to %s\n", name, dst)
		}
	}

	return nil
}
