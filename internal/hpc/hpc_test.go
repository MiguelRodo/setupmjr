package hpc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupHPCRejectsNonLinuxBeforeChangingHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := setupHPC("darwin")
	if err == nil || err.Error() != "HPC setup is Linux-only" {
		t.Fatalf("setupHPC(darwin) error = %v, want Linux-only error", err)
	}

	for _, path := range []string{".bashrc", ".bashrc.d", ".zshrc", ".zshrc.d", ".config"} {
		if _, err := os.Stat(filepath.Join(home, path)); !os.IsNotExist(err) {
			t.Fatalf("setupHPC(darwin) changed %s before failing", path)
		}
	}
}
