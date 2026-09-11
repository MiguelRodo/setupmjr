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

func setupAgentAuth(home, endpoint string) error {
	switch endpoint {
	case "deepseek":
		return setupCopilotDeepSeekAuth(home)
	default:
		return fmt.Errorf("unsupported auth endpoint %q", endpoint)
	}
}

func setupCopilotDeepSeekAuth(home string) error {
	key, err := deepSeekAPIKey(home, filepath.Join(home, ".codex"))
	if err != nil {
		return err
	}

	authDir := filepath.Join(home, ".config", "setupmjr", "agent", "auth")
	if err := os.MkdirAll(authDir, 0700); err != nil {
		return fmt.Errorf("create agent auth directory: %w", err)
	}
	if err := os.Chmod(authDir, 0700); err != nil {
		return fmt.Errorf("secure agent auth directory: %w", err)
	}
	keyPath := deepSeekCredentialPath(home)
	if err := os.WriteFile(keyPath, []byte(key+"\n"), 0600); err != nil {
		return fmt.Errorf("store DeepSeek API key: %w", err)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		return fmt.Errorf("secure DeepSeek API key: %w", err)
	}

	agentDir := filepath.Join(home, ".config", "setupmjr", "agent")
	if err := os.MkdirAll(agentDir, 0700); err != nil {
		return fmt.Errorf("create setupmjr agent directory: %w", err)
	}
	if err := os.Chmod(agentDir, 0700); err != nil {
		return fmt.Errorf("secure setupmjr agent directory: %w", err)
	}
	envPath := copilotEnvPath(home)
	envContent := fmt.Sprintf(`# Managed by setupmjr. DeepSeek API key is loaded from the private credential file.
export COPILOT_PROVIDER_TYPE=%s
export COPILOT_PROVIDER_BASE_URL=%s
export COPILOT_MODEL=%s
export COPILOT_PROVIDER_MAX_PROMPT_TOKENS=%s
export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS=%s
if [ -r "$HOME/.config/setupmjr/agent/auth/deepseek.key" ]; then
    export COPILOT_PROVIDER_API_KEY="$(cat "$HOME/.config/setupmjr/agent/auth/deepseek.key")"
fi
`,
		shellQuote(copilotDeepSeekProviderType),
		shellQuote(copilotDeepSeekBaseURL),
		shellQuote(copilotDeepSeekModel),
		shellQuote(copilotDeepSeekMaxPromptTokens),
		shellQuote(copilotDeepSeekMaxOutputTokens),
	)
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("write Copilot DeepSeek environment: %w", err)
	}
	if err := os.Chmod(envPath, 0600); err != nil {
		return fmt.Errorf("secure Copilot DeepSeek environment: %w", err)
	}

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

	fmt.Println("Configured Copilot CLI to use DeepSeek V4 Pro through DeepSeek's Anthropic-compatible endpoint.")
	fmt.Println("Restart the shell, or source ~/.config/setupmjr/agent/copilot.env, before starting Copilot.")
	return nil
}

func normaliseAuthEndpoint(value string) (string, error) {
	switch strings.ToLower(value) {
	case "d", "deepseek":
		return "deepseek", nil
	default:
		return "", fmt.Errorf("unknown auth endpoint %q; currently supported: deepseek (d)", value)
	}
}

func deepSeekCredentialPath(home string) string {
	return filepath.Join(home, ".config", "setupmjr", "agent", "auth", "deepseek.key")
}

func copilotEnvPath(home string) string {
	return filepath.Join(home, ".config", "setupmjr", "agent", "copilot.env")
}

func currentCopilotEndpoint(home string) string {
	content, err := os.ReadFile(copilotEnvPath(home))
	if err != nil {
		return "default"
	}
	text := string(content)
	if strings.Contains(text, "COPILOT_PROVIDER_BASE_URL="+shellQuote(copilotDeepSeekBaseURL)) {
		return "deepseek"
	}
	return "custom"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
