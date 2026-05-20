package r

import (
	"fmt"
	"path/filepath"

	"github.com/MiguelRodo/setupmjr/internal/assets"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

// SetupRRadian copies .radian_profile and .lintr to the home directory.
func SetupRRadian() error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	dstRadian := filepath.Join(home, ".radian_profile")
	if err := sysutil.CopyEmbedFile(assets.FS, "r/.radian_profile", dstRadian, 0644); err != nil {
		return fmt.Errorf("copy .radian_profile: %w", err)
	}
	fmt.Printf("Copied .radian_profile to %s\n", dstRadian)

	dstLintr := filepath.Join(home, ".lintr")
	if err := sysutil.CopyEmbedFile(assets.FS, "r/.lintr", dstLintr, 0644); err != nil {
		return fmt.Errorf("copy .lintr: %w", err)
	}
	fmt.Printf("Copied .lintr to %s\n", dstLintr)

	return nil
}
