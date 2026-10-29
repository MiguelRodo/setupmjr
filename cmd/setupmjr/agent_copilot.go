package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	copilotDeepSeekProviderType    = "anthropic"
	copilotDeepSeekBaseURL         = "https://api.deepseek.com/anthropic"
	copilotDeepSeekModel           = "deepseek-flash"
	copilotDeepSeekMaxPromptTokens = "840000"
	copilotDeepSeekMaxOutputTokens = "128000"
)

func switchCopilotProvider(home, provider string) error {
	switch provider {
	case "openai":
		return configureCopilotOpenAI(home)
	case "deepseek":
		return configureCopilotDeepSeek(home)
	case "github":
		return configureCopilotGitHub(home)
	default:
		return fmt.Errorf("unsupported Copilot provider %q", provider)
	}
}

func configureCopilotDeepSeek(home string) error {
	if !copilotKeyPresent(home, "deepseek") {
		return fmt.Errorf("DeepSeek credential is not configured; run setupmjr agent --auth deepseek first")
	}

	envContent := copilotCleanEnvironment + fmt.Sprintf(`# Managed by setupmjr. DeepSeek is the active Copilot model provider.
SETUPMJR_COPILOT_ROUTE=deepseek
export COPILOT_PROVIDER_TYPE=%s
export COPILOT_PROVIDER_BASE_URL=%s
export COPILOT_MODEL=%s
export COPILOT_PROVIDER_MAX_PROMPT_TOKENS=%s
export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=%s
if [ -n "${DEEPSEEK_API_KEY:-}" ]; then
    export COPILOT_PROVIDER_API_KEY="$DEEPSEEK_API_KEY"
elif [ -r "$HOME/.config/setupmjr/agent/auth/deepseek.key" ]; then
    export COPILOT_PROVIDER_API_KEY="$(cat "$HOME/.config/setupmjr/agent/auth/deepseek.key")"
fi
if [ -z "${COPILOT_PROVIDER_API_KEY:-}" ]; then
    printf 'DeepSeek key missing; export DEEPSEEK_API_KEY or run setupmjr agent --auth deepseek.\n' >&2
    return 1
fi
`,
		shellQuote(copilotDeepSeekProviderType),
		shellQuote(copilotDeepSeekBaseURL),
		shellQuote(copilotDeepSeekModel),
		shellQuote(copilotDeepSeekMaxPromptTokens),
		shellQuote(copilotDeepSeekMaxOutputTokens),
	)
	if err := writeCopilotEnv(home, envContent); err != nil {
		return err
	}
	if err := wireCopilotEnv(home); err != nil {
		return err
	}

	fmt.Println("Copilot provider switched to DeepSeek Flash using DeepSeek's Anthropic-compatible endpoint.")
	printCopilotShellInstructions()
	return nil
}

func configureCopilotGitHub(home string) error {
	envContent := copilotCleanEnvironment + "# Managed by setupmjr. GitHub-managed Copilot models are active.\nSETUPMJR_COPILOT_ROUTE=github\n"
	if err := writeCopilotEnv(home, envContent); err != nil {
		return err
	}
	if err := wireCopilotEnv(home); err != nil {
		return err
	}

	fmt.Println("Copilot provider switched to GitHub-managed models.")
	printCopilotShellInstructions()
	return nil
}

func writeCopilotEnv(home, content string) error {
	agentDir := filepath.Join(home, ".config", "setupmjr", "agent")
	if err := os.MkdirAll(agentDir, 0700); err != nil {
		return fmt.Errorf("create setupmjr agent directory: %w", err)
	}
	if err := os.Chmod(agentDir, 0700); err != nil {
		return fmt.Errorf("secure setupmjr agent directory: %w", err)
	}

	envPath := copilotEnvPath(home)
	if err := writeCopilotManagedFile(envPath, content); err != nil {
		return fmt.Errorf("write Copilot provider environment: %w", err)
	}
	return nil
}

// Atomic replacement prevents a concurrent launch from sourcing a partial
// selection and falling back to a different route. Never follow file symlinks.
func writeCopilotManagedFile(path, content string) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("managed Copilot path must be a regular file")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".copilot-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(content)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), path)
}

func wireCopilotEnv(home string) error {
	path := filepath.Join(home, ".config", "setupmjr", "agent", "copilot.sh")
	if err := writeCopilotManagedFile(path, copilotShellWrapper); err != nil {
		return fmt.Errorf("write Copilot shell integration: %w", err)
	}
	sourceBlock := `if [ -r "$HOME/.config/setupmjr/agent/copilot.sh" ]; then
    . "$HOME/.config/setupmjr/agent/copilot.sh"
fi`
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if err := updateManagedBlock(
			filepath.Join(home, rc),
			"# setupmjr-copilot-provider:start",
			"# setupmjr-copilot-provider:end",
			sourceBlock,
		); err != nil {
			return fmt.Errorf("configure %s for Copilot provider: %w", rc, err)
		}
	}
	return nil
}

func normaliseCopilotProvider(value string) (string, error) {
	switch strings.ToLower(value) {
	case "o", "openai":
		return "openai", nil
	case "g", "github", "default":
		return "github", nil
	case "d", "deepseek":
		return "deepseek", nil
	default:
		return "", fmt.Errorf("unknown Copilot provider %q; use github (g), deepseek (d) or openai (o)", value)
	}
}

func copilotEnvPath(home string) string {
	return filepath.Join(home, ".config", "setupmjr", "agent", "copilot.env")
}

func currentCopilotProvider(home string) string {
	content, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return "github"
		}
		return "unreadable"
	}
	text := string(content)
	if strings.Contains(text, "COPILOT_PROVIDER_BASE_URL='https://api.openai.com/v1'") {
		return "openai"
	}
	if strings.Contains(text, "COPILOT_PROVIDER_BASE_URL="+shellQuote(copilotDeepSeekBaseURL)) {
		return "deepseek"
	}
	if strings.Contains(text, "unset COPILOT_PROVIDER_TYPE") {
		return "github"
	}
	return "custom"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
