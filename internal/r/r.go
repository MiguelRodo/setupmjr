package r

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

// SetupR copies or appends to .radian_profile and .lintr in the home directory,
// and conditionally configures the switch_r script in the current working directory.
func SetupR(notRadian bool, notLintr bool, switchR bool) error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	if !notRadian {
		radianContent := "options(radian.auto_match = FALSE)\n"
		dstRadian := filepath.Join(home, ".radian_profile")
		if err := sysutil.AppendToFileIfNotExists(dstRadian, radianContent, 0644); err != nil {
			return fmt.Errorf("append to .radian_profile: %w", err)
		}
		fmt.Printf("Configured .radian_profile at %s\n", dstRadian)
	}

	if !notLintr {
		lintrContent := `linters: linters_with_defaults(
    object_length_linter = NULL,
    object_name_linter = NULL)
`
		dstLintr := filepath.Join(home, ".lintr")
		if err := sysutil.AppendToFileIfNotExists(dstLintr, lintrContent, 0644); err != nil {
			return fmt.Errorf("append to .lintr: %w", err)
		}
		fmt.Printf("Configured .lintr at %s\n", dstLintr)
	}

	if switchR {
		if err := setupSwitchR(); err != nil {
			return fmt.Errorf("setup switch_r: %w", err)
		}
	}

	return nil
}

func setupSwitchR() error {
	switchRScript := `
switch_r <- function(wd) {
  if (!dir.exists(wd)) {
    stop("The specified directory does not exist.")
  }

  rm(list = ls(all.names = TRUE, envir = .GlobalEnv), envir = .GlobalEnv)

  setwd(wd)

  if (file.exists(".Renviron")) {
    readRenviron(".Renviron")
  }

  if (file.exists(".Rprofile")) {
    source(".Rprofile", local = .GlobalEnv)
  }

  if (file.exists(".RData")) {
    load(".RData", envir = .GlobalEnv)
  }
  if (interactive() && file.exists(".Rhistory")) {
    loadhistory(".Rhistory")
  }

  if (exists(".First", envir = .GlobalEnv, inherits = FALSE)) {
    do.call(".First", list(), envir = .GlobalEnv)
  }

  invisible(NULL)
}
`
	// 1. Get the current working directory safely
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// 2. Define target directory and file path
	targetDir := filepath.Join(cwd, "scripts", "r")
	dstSwitchRScript := filepath.Join(targetDir, "switch_r.R")

	// 3. Ensure the nested directories (scripts/r) actually exist first
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// 4. Check if the file already exists
	if _, err := os.Stat(dstSwitchRScript); os.IsNotExist(err) {
		// File doesn't exist, so create and write it
		err = os.WriteFile(dstSwitchRScript, []byte(switchRScript), 0644)
		if err != nil {
			return fmt.Errorf("failed to write switch_r.R: %w", err)
		}
		fmt.Printf("Configured switch_r.R at %s\n", dstSwitchRScript)
	} else if err != nil {
		// Some other error occurred (e.g. permission issues)
		return fmt.Errorf("error checking file stats: %w", err)
	} else {
		// File already exists
		fmt.Printf("File already exists at %s, skipping.\n", dstSwitchRScript)
	}

	// 5. Handle .Rprofile top-alignment
	rprofilePath := filepath.Join(cwd, ".Rprofile")
	targetLine := `try(source("scripts/r/switch_r.R"))`

	var existingLines []string

	// Read existing content if file exists
	if _, err := os.Stat(rprofilePath); err == nil {
		content, err := os.ReadFile(rprofilePath)
		if err != nil {
			return fmt.Errorf("failed to read .Rprofile: %w", err)
		}

		// Split file into lines, skipping any existing instances of our target line
		rawLines := strings.Split(string(content), "\n")
		for _, line := range rawLines {
			trimmed := strings.TrimSpace(line)
			if trimmed != targetLine && trimmed != `try(source("scripts/r/switch.R"))` {
				existingLines = append(existingLines, line)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking .Rprofile status: %w", err)
	}

	// Rebuild the file with the target line strictly at the very top
	var newContent bytes.Buffer
	newContent.WriteString(targetLine + "\n")

	// Append back the cleaned remaining lines
	for i, line := range existingLines {
		newContent.WriteString(line)
		// Ensure we don't drop trailing newlines while joining
		if i < len(existingLines)-1 || strings.HasSuffix(line, "\n") {
			newContent.WriteString("\n")
		}
	}

	// Write it back (creates file if it didn't exist)
	if err := os.WriteFile(rprofilePath, newContent.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to update .Rprofile: %w", err)
	}

	fmt.Printf("Successfully enforced '%s' at the top of %s\n", targetLine, rprofilePath)
	return nil
}
