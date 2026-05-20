package gitcmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MiguelRodo/setupmjr/internal/bash"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

// RunCommand runs a command and returns its standard output and error.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func SetupGitUser() error {
	name, _ := RunCommand("git", "config", "--global", "user.name")
	email, _ := RunCommand("git", "config", "--global", "user.email")

	reader := bufio.NewReader(os.Stdin)

	if name == "" {
		fmt.Print("Enter Git user.name: ")
		nameInput, _ := reader.ReadString('\n')
		nameInput = strings.TrimSpace(nameInput)
		if nameInput != "" {
			if _, err := RunCommand("git", "config", "--global", "user.name", nameInput); err != nil {
				return fmt.Errorf("set git user.name: %w", err)
			}
			fmt.Printf("Set git user.name to %s\n", nameInput)
		}
	} else {
		fmt.Printf("Git user.name already set to %s\n", name)
	}

	if email == "" {
		fmt.Print("Enter Git user.email: ")
		emailInput, _ := reader.ReadString('\n')
		emailInput = strings.TrimSpace(emailInput)
		if emailInput != "" {
			if _, err := RunCommand("git", "config", "--global", "user.email", emailInput); err != nil {
				return fmt.Errorf("set git user.email: %w", err)
			}
			fmt.Printf("Set git user.email to %s\n", emailInput)
		}
	} else {
		fmt.Printf("Git user.email already set to %s\n", email)
	}

	return nil
}

func SetupGitLoginText() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter GitHub username: ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Print("Enter GitHub token: ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)

	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	if err := bash.SetupBashRCD(); err != nil {
		return err
	}

	// Make sure login.sh exists
	if err := bash.SetupBashLogin(); err != nil {
		return err
	}

	loginPath := filepath.Join(home, ".bashrc.d", "login.sh")

	if user != "" || token != "" {
		b, err := os.ReadFile(loginPath)
		if err != nil {
			return fmt.Errorf("read login.sh: %w", err)
		}
		content := string(b)

		if user != "" {
			content = injectVar(content, "GITHUB_USERNAME", user)
			content = injectVar(content, "GITHUB_USER", user)
		}
		if token != "" {
			content = injectVar(content, "GH_TOKEN", token)
			content = injectVar(content, "GITHUB_TOKEN", token)
			content = injectVar(content, "GITHUB_PAT", token)
		}

		if err := os.WriteFile(loginPath, []byte(content), 0755); err != nil {
			return fmt.Errorf("write login.sh: %w", err)
		}
		fmt.Println("Injected credentials into login.sh")
	}

	return nil
}

func injectVar(content, varName, val string) string {
	pattern := regexp.MustCompile(`(?m)^#?\s*` + regexp.QuoteMeta(varName) + `=.*$`)
	replacement := fmt.Sprintf(`%s="%s"`, varName, val)

	if pattern.MatchString(content) {
		return pattern.ReplaceAllString(content, replacement)
	}

	// If not found, append it
	return content + "\n" + replacement + "\n"
}

func SetupGitLoginCache() error {
	if _, err := RunCommand("git", "config", "--global", "credential.helper", "cache"); err != nil {
		return fmt.Errorf("set credential.helper cache: %w", err)
	}
	fmt.Println("Configured git to use cache credential helper")
	return nil
}

func SetupGitLoginMngr() error {
	// Simple approach: we'll just try to set 'manager'
	if _, err := RunCommand("git", "config", "--global", "credential.helper", "manager"); err != nil {
		fmt.Printf("Warning: failed to set manager credential helper: %v\n", err)
	} else {
		fmt.Println("Configured git to use manager credential helper")
	}
	return nil
}

func SetupGitLogin() error {
	if err := SetupGitLoginText(); err != nil {
		return err
	}
	if err := SetupGitLoginCache(); err != nil {
		return err
	}
	if err := SetupGitLoginMngr(); err != nil {
		return err
	}
	return nil
}

func SetupGit() error {
	if err := SetupGitUser(); err != nil {
		return err
	}
	if err := SetupGitLogin(); err != nil {
		return err
	}
	return nil
}
