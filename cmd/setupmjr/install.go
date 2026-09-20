package main

import (
	"flag"
	"fmt"

	"github.com/MiguelRodo/setupmjr/internal/repo"
)

var installRepos = repo.SetupRepoInstallRepos

func handleInstall(args []string) error {
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
		return fmt.Errorf("install requires --repos, --pj, and/or --gh-skill")
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
