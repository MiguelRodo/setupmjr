package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/bash"
	"github.com/MiguelRodo/setupmjr/internal/gitcmd"
	"github.com/MiguelRodo/setupmjr/internal/hpc"
	"github.com/MiguelRodo/setupmjr/internal/r"
	"github.com/MiguelRodo/setupmjr/internal/repo"
	"github.com/MiguelRodo/setupmjr/internal/zsh"
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
	case "repo":
		if err := handleRepo(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Printf("setupmjr version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
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
  hpc scratch
  hpc apptainer
  hpc slurm
  hpc git
  hpc r
  bash rc.d
  bash path
  bash login
  r [--not-radian] [--not-lintr]
  git
  git profile
  git auth
  git auth text
  git auth cache
  git auth mngr
  repo readme
  repo devcontainer [--repo <owner/repo>@<branch>] [--build]
  repo action <action-name>
  repo install repos`)
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
		if err := bash.SetupBashPath(); err != nil {
			return err
		}
		if err := bash.SetupBashLogin(); err != nil {
			return err
		}
		if err := zsh.SetupZshRCD(); err != nil {
			return err
		}
		if err := zsh.SetupZshLogin(); err != nil {
			return err
		}
		// Execute SetupHPCGit AFTER SetupBashLogin and SetupZshLogin so login.sh is not overwritten
		if err := hpc.SetupHPCGit(); err != nil {
			return err
		}
		if err := hpc.SetupHPCScratch(); err != nil {
			return err
		}
		if err := hpc.SetupHPCApptainer(); err != nil {
			return err
		}
		if err := hpc.SetupHPCSlurm(); err != nil {
			return err
		}
		if err := handleR([]string{}); err != nil {
			return err
		}
		fmt.Println("Please run 'source ~/.bashrc' to apply the changes.")
		return nil
	}

	subcmd := args[0]
	var err error
	switch subcmd {
	case "scratch":
		err = hpc.SetupHPCScratch()
	case "apptainer":
		err = hpc.SetupHPCApptainer()
	case "slurm":
		err = hpc.SetupHPCSlurm()
	case "git":
		err = hpc.SetupHPCGit()
	case "r":
		err = handleR(args[1:])
	default:
		return fmt.Errorf("unknown hpc subcommand: %s", subcmd)
	}

	if err == nil {
		fmt.Println("Please run 'source ~/.bashrc' to apply the changes.")
	}

	return err
}

func handleBash(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bash requires a subcommand: rc.d, path, or login")
	}

	subcmd := args[0]
	switch subcmd {
	case "rc.d":
		return bash.SetupBashRCD()
	case "path":
		return bash.SetupBashPath()
	case "login":
		return bash.SetupBashLogin()
	default:
		return fmt.Errorf("unknown bash subcommand: %s", subcmd)
	}
}

func handleR(args []string) error {
	fs := flag.NewFlagSet("r", flag.ExitOnError)
	notRadian := fs.Bool("not-radian", false, "Do not configure radian")
	notLintr := fs.Bool("not-lintr", false, "Do not configure lintr")
	fs.Parse(args)

	return r.SetupR(*notRadian, *notLintr)
}

func handleGit(args []string) error {
	if len(args) == 0 {
		return gitcmd.SetupGit()
	}

	subcmd := args[0]
	switch subcmd {
	case "profile":
		return gitcmd.SetupGitProfile()
	case "auth":
		if len(args) == 1 {
			return gitcmd.SetupGitAuth()
		}
		loginSubcmd := args[1]
		switch loginSubcmd {
		case "text":
			return gitcmd.SetupGitAuthText()
		case "cache":
			return gitcmd.SetupGitAuthCache()
		case "mngr":
			return gitcmd.SetupGitAuthMngr()
		default:
			return fmt.Errorf("unknown git auth subcommand: %s", loginSubcmd)
		}
	default:
		return fmt.Errorf("unknown git subcommand: %s", subcmd)
	}
}

func handleRepo(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("repo requires a subcommand: readme, devcontainer, action, or install repos")
	}

	subcmd := args[0]
	switch subcmd {
	case "readme":
		return repo.SetupRepoReadme()
	case "devcontainer":
		fs := flag.NewFlagSet("devcontainer", flag.ExitOnError)
		repoFlag := fs.String("repo", "MiguelRodo/comp@main", "Repository and branch to copy .devcontainer from, formatted as <owner/repo>@<branch>")
		buildFlag := fs.Bool("build", false, "Copy devcontainer prebuild action")
		fs.Parse(args[1:])

		repoStr := *repoFlag
		branchStr := "main"
		if strings.Contains(repoStr, "@") {
			parts := strings.SplitN(repoStr, "@", 2)
			repoStr = parts[0]
			branchStr = parts[1]
		}

		return repo.SetupRepoDevcontainer(repoStr, branchStr, *buildFlag)
	case "action":
		if len(args) < 2 {
			return fmt.Errorf("repo action requires an action name")
		}
		return repo.SetupRepoAction(args[1])
	case "install":
		if len(args) > 1 && args[1] == "repos" {
			return repo.SetupRepoInstallRepos()
		}
		return fmt.Errorf("unknown repo install subcommand")
	default:
		return fmt.Errorf("unknown repo subcommand: %s", subcmd)
	}
}
