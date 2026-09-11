package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func setupAgentAuth(home, endpoint string) error {
	switch endpoint {
	case "deepseek":
		return setupDeepSeekAuth(home)
	default:
		return fmt.Errorf("unsupported auth endpoint %q", endpoint)
	}
}

func setupDeepSeekAuth(home string) error {
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

	fmt.Println("Stored DeepSeek API credential for setupmjr-managed agents.")
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
