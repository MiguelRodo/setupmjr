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
	copilotDeepSeekModel           = "deepseek-v4-pro"
	copilotDeepSeekMaxPromptTokens = "840000"
	copilotDeepSeekMaxOutputTokens = "128000"
)

func switchCopilotProvider(home, provider string) error {
	switch provider {
	case "deepseek":
		return configureCopilotDeepSeek(home)
	case "github":
		return configureCopilotGitHub(home)
	default:
		return fmt.Errorf("unsupported Copilot provider %q", provider)
	}
}

func configureCopilotDeepSeek(home string) error {
	keyBytes, err := os.ReadFile(deepSeekCredentialPath(home))
	if err != nil || strings.TrimSpace(string(keyBytes)) == "" {
		return fmt.Errorf("DeepSeek credential is not configured; run setupmjr agent --auth deepseek first")
	}

	envContent := fmt.Sprintf(`# Managed by setupmjr. DeepSeek is the active Copilot model provider.
export COPILOT_PROVIDER_TYPE=%s
export COPILOT_PROVIDER_BASE_URL=%s
export COPILOT_MODEL=%s
export COPILOT_PROVIDER_MAX_PROMPT_TOKENS=%s
export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=%s
export COPILOT_PROVIDER_API_KEY="$(cat "$HOME/.config/setupmjr/agent/auth/deepseek.key")"
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

	fmt.Println("Copilot provider switched to DeepSeek V4 Pro using DeepSeek's Anthropic-compatible endpoint.")
	fmt.Println("Restart the shell, or source ~/.config/setupmjr/agent/copilot.env, before starting Copilot.")
	return nil
}

func configureCopilotGitHub(home string) error {
	envContent := `# Managed by setupmjr. GitHub-managed Copilot models are active.
unset COPILOT_PROVIDER_TYPE
unset COPILOT_PROVIDER_BASE_URL
unset COPILOT_PROVIDER_API_KEY
unset COPILOT_MODEL
unset COPILOT_PROVIDER_MAX_PROMPT_TOKENS
unset COPILOT_PROVIDER_MAX_OUTPUT_TOKENS
`
	if err := writeCopilotEnv(home, envContent); err != nil {
		return err
	}
	if err := wireCopilotEnv(home); err != nil {
		return err
	}

	fmt.Println("Copilot provider switched to GitHub-managed models.")
	fmt.Println("Restart the shell, or source ~/.config/setupmjr/agent/copilot.env, before starting Copilot.")
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
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("write Copilot provider environment: %w", err)
	}
	if err := os.Chmod(envPath, 0600); err != nil {
		return fmt.Errorf("secure Copilot provider environment: %w", err)
	}
	return nil
}

func wireCopilotEnv(home string) error {
	sourceBlock := `if [ -r "$HOME/.config/setupmjr/agent/copilot.env" ]; then
    . "$HOME/.config/setupmjr/agent/copilot.env"
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
	case "g", "github", "default":
		return "github", nil
	case "d", "deepseek":
		return "deepseek", nil
	default:
		return "", fmt.Errorf("unknown Copilot provider %q; use github (g) or deepseek (d)", value)
	}
}

func copilotEnvPath(home string) string {
	return filepath.Join(home, ".config", "setupmjr", "agent", "copilot.env")
}

func currentCopilotProvider(home string) string {
	content, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		return "github"
	}
	text := string(content)
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
