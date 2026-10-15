package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/repo"
)

var installRepos = repo.SetupRepoInstallRepos

func handleInstall(args []string) error {
	if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		printInstallUsage()
		return nil
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		if len(args) != 1 {
			return fmt.Errorf("install %s does not accept additional arguments", args[0])
		}
		switch args[0] {
		case "claude", "codex", "copilot":
			if err := installCodingAgent(args[0]); err != nil {
				return err
			}
			fmt.Printf("%s installer completed. Restart your shell if needed; sign-in is separate.\n", args[0])
			return nil
		default:
			return fmt.Errorf("unknown install subcommand %q (choose claude, codex or copilot)", args[0])
		}
	}
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	reposFlag := fs.Bool("repos", false, "Install or update the repos CLI")
	pj := fs.Bool("pj", false, "Install or update the pj launcher from MiguelRodo/pj")
	ghSkill := fs.Bool("gh-skill", false, "Install the github-projects skill for universal agents at user scope")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("install does not accept positional arguments")
	}
	if !*reposFlag && !*pj && !*ghSkill {
		return fmt.Errorf("install requires claude, codex, copilot, --repos, --pj, or --gh-skill")
	}

	if *reposFlag {
		if err := installRepos(); err != nil {
			return fmt.Errorf("install repos: %w", err)
		}
	}
	if *ghSkill {
		if err := installGhSkill(); err != nil {
			return fmt.Errorf("install github-projects skill: %w", err)
		}
		fmt.Println("Installed github-projects at universal user scope.")
	}
	if *pj {
		if err := installPj(); err != nil {
			return fmt.Errorf("install pj launcher: %w", err)
		}
		fmt.Println("Installed or updated pj launcher from canonical source MiguelRodo/pj.")
	}
	return nil
}

func printInstallUsage() {
	fmt.Println(`Usage:
  setupmjr install <claude|codex|copilot>
  setupmjr install [--repos] [--gh-skill] [--pj]

The coding-agent subcommands run vendor-supported CLI installers.
They do not sign you in or change your existing agent provider configuration.
Legacy install flags may be combined; agent subcommands are separate.`)
}
