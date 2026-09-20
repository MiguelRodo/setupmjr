package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/gitcmd"
	"github.com/MiguelRodo/setupmjr/internal/hpc"
	"github.com/MiguelRodo/setupmjr/internal/r"
	"github.com/MiguelRodo/setupmjr/internal/repo"
	"github.com/MiguelRodo/setupmjr/internal/shell"
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
	case "shell":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: shell requires a shell name (e.g., bash, zsh)\n")
			os.Exit(1)
		}
		if err := handleShellCmd(os.Args[3:], os.Args[2]); err != nil {
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
	case "install":
		if err := handleInstall(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "agent":
		if err := handleAgent(os.Args[2:]); err != nil {
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
  shell <shell> rc.d
  shell <shell> path
  shell <shell> login
  bash rc.d
  bash path
  bash login
  r [--not-radian] [--not-lintr] [--switch-r]
  git
  git profile
  git auth
  git auth text
  git auth cache
  git auth mngr
  repo readme
  repo devcontainer [--repo <owner/repo>@<branch>] [--build]
  repo action <action-name>
  install [--repos] [--pj] [--gh-skill]
  agent [-a <endpoint>] [-p [g|d]] [-c [o|d]] [-s [agy|deepseek]] [--config] [--list]`)
}

func handleHPC(args []string) error {
	if len(args) == 0 {
		if err := hpc.SetupHPC(); err != nil {
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
		err = gitcmd.SetupGit()
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
	return handleShellCmd(args, "bash")
}

func handleShellCmd(args []string, shellName string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires a subcommand: rc.d, path, or login", shellName)
	}

	subcmd := args[0]
	switch subcmd {
	case "rc.d":
		return shell.SetupShellRCD(shellName)
	case "path":
		return shell.SetupShellPath(shellName)
	case "login":
		fs := flag.NewFlagSet("login", flag.ExitOnError)
		notProfile := fs.Bool("not-profile", false, "Do not configure profile files (~/.profile, etc.)")
		fs.Parse(args[1:])

		return shell.SetupShellLogin(shellName, *notProfile)
	default:
		return fmt.Errorf("unknown %s subcommand: %s", shellName, subcmd)
	}
}

func handleR(args []string) error {
	fs := flag.NewFlagSet("r", flag.ExitOnError)
	notRadian := fs.Bool("not-radian", false, "Do not configure radian")
	notLintr := fs.Bool("not-lintr", false, "Do not configure lintr")
	switchR := fs.Bool("switch-r", false, "Configure the switch_r script and .Rprofile")
	fs.Parse(args)

	return r.SetupR(*notRadian, *notLintr, *switchR)
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
		fs := flag.NewFlagSet("auth", flag.ExitOnError)
		system := fs.Bool("system", false, "Use system git config")
		local := fs.Bool("local", false, "Use local git config")
		remove := fs.Bool("remove", false, "Remove existing credential helpers in the selected scope")

		loginSubcmd := ""
		authArgs := args[1:]

		if len(authArgs) > 0 && !strings.HasPrefix(authArgs[0], "-") {
			loginSubcmd = authArgs[0]
			fs.Parse(authArgs[1:])
		} else {
			fs.Parse(authArgs)
			if len(fs.Args()) > 0 {
				loginSubcmd = fs.Args()[0]
			}
		}

		scope := "--global"
		if *system {
			scope = "--system"
		} else if *local {
			scope = "--local"
		}

		if *remove {
			gitcmd.RunCommand("git", "config", scope, "--unset-all", "credential.helper")
			gitcmd.RunCommand("git", "config", scope, "--unset-all", "credential.https://github.com.helper")
		}

		if loginSubcmd == "" {
			return gitcmd.SetupGitAuth(scope)
		}

		switch loginSubcmd {
		case "text":
			return gitcmd.SetupGitAuthText(scope)
		case "cache":
			return gitcmd.SetupGitAuthCache(scope)
		case "mngr":
			return gitcmd.SetupGitAuthMngr(scope)
		default:
			return fmt.Errorf("unknown git auth subcommand: %s", loginSubcmd)
		}
	default:
		return fmt.Errorf("unknown git subcommand: %s", subcmd)
	}
}

func handleRepo(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("repo requires a subcommand: readme, devcontainer, or action")
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
	default:
		return fmt.Errorf("unknown repo subcommand: %s", subcmd)
	}
}
