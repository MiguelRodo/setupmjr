package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/MiguelRodo/setupmjr/internal/bash"
	"github.com/MiguelRodo/setupmjr/internal/gitcmd"
	"github.com/MiguelRodo/setupmjr/internal/hpc"
	"github.com/MiguelRodo/setupmjr/internal/r"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "hpc":
		if err := handleHPC(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "bash":
		if err := handleBash(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "r":
		if err := handleR(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "git":
		if err := handleGit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("setupmjr version %s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`setupmjr - Cross-platform setup utility
Commands:
  hpc       Master HPC setup
  hpc scratch [--r]
  hpc apptainer
  hpc slurm
  hpc login git
  hpc git
  bash rc.d
  bash login
  r radian
  git
  git user
  git login
  git login text
  git login cache
  git login mngr`)
}

func handleHPC(args []string) error {
	if len(args) == 0 {
		// Run master hpc setup
		if err := hpc.SetupHPC(); err != nil {
			return err
		}
		if err := bash.SetupBashRCD(); err != nil {
			return err
		}
		if err := bash.SetupBashLogin(); err != nil {
			return err
		}
		// Execute SetupHPCGit AFTER SetupBashLogin so login.sh is not overwritten
		if err := hpc.SetupHPCGit(); err != nil {
			return err
		}
		if err := hpc.SetupHPCScratch(false); err != nil {
			return err
		}
		if err := hpc.SetupHPCApptainer(); err != nil {
			return err
		}
		if err := hpc.SetupHPCSlurm(); err != nil {
			return err
		}
		return nil
	}

	subcmd := args[0]
	switch subcmd {
	case "scratch":
		fs := flag.NewFlagSet("scratch", flag.ExitOnError)
		withR := fs.Bool("r", false, "Include R config")
		fs.Parse(args[1:])
		return hpc.SetupHPCScratch(*withR)
	case "apptainer":
		return hpc.SetupHPCApptainer()
	case "slurm":
		return hpc.SetupHPCSlurm()
	case "git":
		return hpc.SetupHPCGit()
	case "login":
		if len(args) > 1 && args[1] == "git" {
			return hpc.SetupHPCLoginGit()
		}
		return fmt.Errorf("unknown hpc login subcommand")
	default:
		return fmt.Errorf("unknown hpc subcommand: %s", subcmd)
	}
}

func handleBash(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bash requires a subcommand: rc.d or login")
	}

	subcmd := args[0]
	switch subcmd {
	case "rc.d":
		return bash.SetupBashRCD()
	case "login":
		return bash.SetupBashLogin()
	default:
		return fmt.Errorf("unknown bash subcommand: %s", subcmd)
	}
}

func handleR(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("r requires a subcommand: radian")
	}

	subcmd := args[0]
	switch subcmd {
	case "radian":
		return r.SetupRRadian()
	default:
		return fmt.Errorf("unknown r subcommand: %s", subcmd)
	}
}

func handleGit(args []string) error {
	if len(args) == 0 {
		return gitcmd.SetupGit()
	}

	subcmd := args[0]
	switch subcmd {
	case "user":
		return gitcmd.SetupGitUser()
	case "login":
		if len(args) == 1 {
			return gitcmd.SetupGitLogin()
		}
		loginSubcmd := args[1]
		switch loginSubcmd {
		case "text":
			return gitcmd.SetupGitLoginText()
		case "cache":
			return gitcmd.SetupGitLoginCache()
		case "mngr":
			return gitcmd.SetupGitLoginMngr()
		default:
			return fmt.Errorf("unknown git login subcommand: %s", loginSubcmd)
		}
	default:
		return fmt.Errorf("unknown git subcommand: %s", subcmd)
	}
}
