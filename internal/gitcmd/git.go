package gitcmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/MiguelRodo/setupmjr/internal/shell"
	"github.com/MiguelRodo/setupmjr/internal/sysutil"
)

const (
	githubHost       = "github.com"
	githubHelper     = `!f() { if [ "$1" = "get" ]; then token="$(cat "$HOME/.config/setupmjr/auth/github.token" 2>/dev/null)" || exit 1; printf 'username=x-access-token\npassword=%s\n' "$token"; fi; }; f`
	githubBlockStart = "# setupmjr-github-token:start"
	githubBlockEnd   = "# setupmjr-github-token:end"
)

// RunCommand runs a command and returns its standard output and error.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func SetupGitProfile() error {
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

// SetupGitAuth chooses the least custom durable GitHub authentication path.
// An existing persistent gh login is preferred. Otherwise setupmjr keeps one
// protected token file as a headless/HPC fallback.
func SetupGitAuth(scope string) error {
	if ghStoredAuthAvailable() {
		return SetupGitAuthGh(scope)
	}
	return SetupGitAuthText(scope)
}

// SetupGitAuthGh configures Git to use an existing GitHub CLI login. Environment
// tokens are ignored for the authentication check so this path really represents
// gh-owned persistent authentication rather than an ephemeral GH_TOKEN.
func SetupGitAuthGh(scope string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("GitHub CLI (gh) is not installed")
	}
	if !ghStoredAuthAvailable() {
		return fmt.Errorf("GitHub CLI is not persistently authenticated for %s; run 'gh auth login' or use 'setupmjr git auth text'", githubHost)
	}

	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}
	if err := scrubLegacyGitHubCredentials(home); err != nil {
		return err
	}
	if err := removeGitHubTokenShellWiring(home); err != nil {
		return err
	}

	if scope == "--global" {
		cmd := exec.Command("gh", "auth", "setup-git", "--hostname", githubHost)
		cmd.Env = withoutGitHubTokenEnv(os.Environ())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("configure Git to use GitHub CLI credentials: %w", err)
		}
	} else {
		if _, err := RunCommand("git", "config", scope, "--replace-all", "credential.https://github.com.helper", "!gh auth git-credential"); err != nil {
			return fmt.Errorf("configure GitHub CLI credential helper: %w", err)
		}
	}

	fmt.Println("Configured Git to use the existing GitHub CLI login.")
	return nil
}

// SetupGitAuthText keeps the headless/HPC fallback while storing the PAT only
// once. The old command name remains for compatibility, but the token itself is
// no longer embedded in login.sh or any shell startup file.
func SetupGitAuthText(scope string) error {
	home, err := sysutil.HomeDir()
	if err != nil {
		return err
	}

	token, err := resolveGitHubToken(home)
	if err != nil {
		return err
	}
	if err := storeGitHubToken(home, token); err != nil {
		return err
	}
	if err := scrubLegacyGitHubCredentials(home); err != nil {
		return err
	}
	if err := wireGitHubTokenEnv(home); err != nil {
		return err
	}
	if _, err := RunCommand("git", "config", scope, "--replace-all", "credential.https://github.com.helper", githubHelper); err != nil {
		return fmt.Errorf("configure GitHub credential helper: %w", err)
	}

	fmt.Printf("Configured GitHub authentication using %s\n", githubTokenPath(home))
	return nil
}

func resolveGitHubToken(home string) (string, error) {
	if token := environmentGitHubToken(); token != "" {
		return token, nil
	}

	if tokenBytes, err := os.ReadFile(githubTokenPath(home)); err == nil {
		if token := strings.TrimSpace(string(tokenBytes)); token != "" {
			return token, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read GitHub token: %w", err)
	}

	fmt.Fprint(os.Stderr, "GitHub token: ")
	var tokenBytes []byte
	var err error
	if term.IsTerminal(int(os.Stdin.Fd())) {
		tokenBytes, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
	} else {
		line, readErr := bufio.NewReader(os.Stdin).ReadString('\n')
		tokenBytes = []byte(line)
		err = readErr
		if err == io.EOF && len(tokenBytes) > 0 {
			err = nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("read GitHub token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return "", fmt.Errorf("GitHub token cannot be empty")
	}
	return token, nil
}

func environmentGitHubToken() string {
	if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		return token
	}
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
}

func githubTokenPath(home string) string {
	return filepath.Join(home, ".config", "setupmjr", "auth", "github.token")
}

func storeGitHubToken(home, token string) error {
	dir := filepath.Dir(githubTokenPath(home))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create setupmjr auth directory: %w", err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("secure setupmjr auth directory: %w", err)
	}
	path := githubTokenPath(home)
	if err := os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
		return fmt.Errorf("write GitHub token: %w", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("secure GitHub token: %w", err)
	}
	return nil
}

func wireGitHubTokenEnv(home string) error {
	block := `if [ -z "${GH_TOKEN:-}" ] && [ -r "$HOME/.config/setupmjr/auth/github.token" ]; then
    export GH_TOKEN="$(cat "$HOME/.config/setupmjr/auth/github.token")"
fi`

	for _, shellName := range []string{"bash", "zsh"} {
		// Migrate the short-lived direct rc wiring from setupmjr #53.
		rcPath := filepath.Join(home, fmt.Sprintf(".%src", shellName))
		if err := removeManagedBlock(rcPath, githubBlockStart, githubBlockEnd); err != nil {
			return fmt.Errorf("remove legacy %s GitHub token environment: %w", rcPath, err)
		}

		if err := shell.SetupShellRCD(shellName); err != nil {
			return err
		}
		if err := shell.SetupShellLogin(shellName, false); err != nil {
			return err
		}

		loginPath := filepath.Join(home, fmt.Sprintf(".%src.d", shellName), "login.sh")
		if err := upsertManagedBlock(loginPath, githubBlockStart, githubBlockEnd, block); err != nil {
			return fmt.Errorf("configure %s GitHub token environment: %w", loginPath, err)
		}
	}
	return nil
}

func removeGitHubTokenShellWiring(home string) error {
	for _, shellName := range []string{"bash", "zsh"} {
		paths := []string{
			filepath.Join(home, fmt.Sprintf(".%src", shellName)),
			filepath.Join(home, fmt.Sprintf(".%src.d", shellName), "login.sh"),
		}
		for _, path := range paths {
			if err := removeManagedBlock(path, githubBlockStart, githubBlockEnd); err != nil {
				return fmt.Errorf("remove %s GitHub token environment: %w", path, err)
			}
		}
	}
	return nil
}

func scrubLegacyGitHubCredentials(home string) error {
	for _, path := range []string{
		filepath.Join(home, ".bashrc.d", "login.sh"),
		filepath.Join(home, ".zshrc.d", "login.sh"),
	} {
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read legacy credential file %s: %w", path, err)
		}

		lines := strings.Split(string(content), "\n")
		cleaned := make([]string, 0, len(lines))
		changed := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "GH_TOKEN=") || strings.HasPrefix(trimmed, "GITHUB_USERNAME=") {
				changed = true
				continue
			}
			cleaned = append(cleaned, line)
		}
		if changed {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(strings.Join(cleaned, "\n")), info.Mode().Perm()); err != nil {
				return fmt.Errorf("remove legacy GitHub credentials from %s: %w", path, err)
			}
		}
	}
	return nil
}

func upsertManagedBlock(path, startMarker, endMarker, block string) error {
	contentBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(contentBytes)
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if (start >= 0) != (end >= 0) {
		return fmt.Errorf("malformed managed block")
	}

	managed := startMarker + "\n" + block + "\n" + endMarker
	if start >= 0 {
		end += len(endMarker)
		content = content[:start] + managed + content[end:]
	} else {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += managed + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func removeManagedBlock(path, startMarker, endMarker string) error {
	contentBytes, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	content := string(contentBytes)
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)
	if start < 0 && end < 0 {
		return nil
	}
	if start < 0 || end < 0 || end < start {
		return fmt.Errorf("malformed managed block")
	}
	end += len(endMarker)
	for end < len(content) && content[end] == '\n' {
		end++
	}
	content = content[:start] + content[end:]
	return os.WriteFile(path, []byte(content), 0644)
}

func ghStoredAuthAvailable() bool {
	if _, err := exec.LookPath("gh"); err != nil {
		return false
	}
	cmd := exec.Command("gh", "auth", "status", "--hostname", githubHost)
	cmd.Env = withoutGitHubTokenEnv(os.Environ())
	return cmd.Run() == nil
}

func withoutGitHubTokenEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, item := range env {
		if strings.HasPrefix(item, "GH_TOKEN=") || strings.HasPrefix(item, "GITHUB_TOKEN=") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func SetupGitAuthCache(scope string) error {
	if _, err := RunCommand("git", "config", scope, "credential.helper", "cache"); err != nil {
		return fmt.Errorf("set credential.helper cache: %w", err)
	}
	fmt.Println("Configured git to use cache credential helper")
	return nil
}

func SetupGitAuthMngr(scope string) error {
	if _, err := RunCommand("git", "config", scope, "credential.helper", "manager"); err != nil {
		fmt.Printf("Warning: failed to set manager credential helper: %v\n", err)
	} else {
		fmt.Println("Configured git to use manager credential helper")
	}
	return nil
}

func SetupGit() error {
	if err := SetupGitProfile(); err != nil {
		return err
	}
	return SetupGitAuth("--global")
}
