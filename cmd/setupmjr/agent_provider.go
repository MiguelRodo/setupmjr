package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const deepSeekCodexSetupURL = "https://cdn.deepseek.com/api-docs/codex-deepseek-setup-en.sh"

var runDeepSeekSetup = runDeepSeekOfficialInstaller

func currentCodexProvider(home string) (string, error) {
	configPath := filepath.Join(home, ".codex", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "openai", nil
		}
		return "", fmt.Errorf("read Codex config: %w", err)
	}
	text := string(content)
	if hasTomlStringValue(text, "model_provider", "deepseek") {
		return "deepseek", nil
	}
	return "openai", nil
}

func hasTomlStringValue(content, key, want string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			// A bare key after a table header belongs to that table, not the root.
			break
		}
		if strings.HasPrefix(trimmed, "#") || !strings.HasPrefix(trimmed, key) {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) != key {
			continue
		}
		value := strings.TrimSpace(strings.SplitN(parts[1], "#", 2)[0])
		unquoted, err := strconv.Unquote(value)
		if err == nil && unquoted == want {
			return true
		}
	}
	return false
}

func switchCodexProvider(home, provider string) error {
	current, err := currentCodexProvider(home)
	if err != nil {
		return err
	}
	if current == provider {
		fmt.Printf("Codex is already configured for %s.\n", provider)
		return nil
	}

	switch provider {
	case "deepseek":
		if current == "openai" {
			if err := saveCodexProviderSnapshot(home, "openai"); err != nil {
				return err
			}
		}
		if codexProviderSnapshotExists(home, "deepseek") {
			if err := restoreCodexProviderSnapshot(home, "deepseek"); err != nil {
				return err
			}
		} else {
			if err := runDeepSeekSetup(home, filepath.Join(home, ".codex"), "1"); err != nil {
				return fmt.Errorf("configure DeepSeek for Codex: %w", err)
			}
			if err := saveCodexProviderSnapshot(home, "deepseek"); err != nil {
				return err
			}
		}
		fmt.Println("Codex provider switched to DeepSeek. Restart active Codex clients before using the new provider.")
		return nil

	case "openai":
		if current == "deepseek" {
			if err := saveCodexProviderSnapshot(home, "deepseek"); err != nil {
				return err
			}
		}
		if codexProviderSnapshotExists(home, "openai") {
			if err := restoreCodexProviderSnapshot(home, "openai"); err != nil {
				return err
			}
		} else {
			// This handles installations that were switched to DeepSeek before
			// setupmjr started managing provider snapshots.
			if err := runDeepSeekSetup(home, filepath.Join(home, ".codex"), "9"); err != nil {
				return fmt.Errorf("restore OpenAI Codex configuration: %w", err)
			}
			if err := saveCodexProviderSnapshot(home, "openai"); err != nil {
				return err
			}
		}
		fmt.Println("Codex provider switched to OpenAI/ChatGPT. Restart active Codex clients before using the restored provider.")
		return nil
	default:
		return fmt.Errorf("unsupported Codex provider %q", provider)
	}
}

func codexStateDir(home string) string {
	return filepath.Join(home, ".config", "setupmjr", "agent", "codex")
}

func codexProviderSnapshotExists(home, provider string) bool {
	_, err := os.Stat(filepath.Join(codexStateDir(home), provider+"-config.toml"))
	return err == nil
}

func saveCodexProviderSnapshot(home, provider string) error {
	stateDir := codexStateDir(home)
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return fmt.Errorf("create setupmjr Codex state directory: %w", err)
	}
	codexHome := filepath.Join(home, ".codex")
	configPath := filepath.Join(codexHome, "config.toml")
	configSnapshot := filepath.Join(stateDir, provider+"-config.toml")
	if _, err := os.Stat(configPath); err == nil {
		if err := copyFile(configPath, configSnapshot, 0600); err != nil {
			return fmt.Errorf("save %s Codex config snapshot: %w", provider, err)
		}
	} else if os.IsNotExist(err) {
		if err := os.Remove(configSnapshot); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else {
		return fmt.Errorf("stat Codex config: %w", err)
	}

	modelsPath := filepath.Join(codexHome, "models.json")
	modelsSnapshot := filepath.Join(stateDir, provider+"-models.json")
	modelsState := filepath.Join(stateDir, provider+"-models.state")
	if _, err := os.Stat(modelsPath); err == nil {
		if err := copyFile(modelsPath, modelsSnapshot, 0600); err != nil {
			return fmt.Errorf("save %s Codex model catalogue: %w", provider, err)
		}
		if err := os.WriteFile(modelsState, []byte("present\n"), 0600); err != nil {
			return fmt.Errorf("save %s model catalogue state: %w", provider, err)
		}
	} else if os.IsNotExist(err) {
		_ = os.Remove(modelsSnapshot)
		if err := os.WriteFile(modelsState, []byte("absent\n"), 0600); err != nil {
			return fmt.Errorf("save %s model catalogue state: %w", provider, err)
		}
	} else {
		return fmt.Errorf("stat Codex model catalogue: %w", err)
	}
	return nil
}

func restoreCodexProviderSnapshot(home, provider string) error {
	stateDir := codexStateDir(home)
	configSnapshot := filepath.Join(stateDir, provider+"-config.toml")
	codexHome := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		return fmt.Errorf("create Codex home: %w", err)
	}
	if _, err := os.Stat(configSnapshot); err != nil {
		return fmt.Errorf("no saved %s Codex configuration", provider)
	}
	if err := copyFile(configSnapshot, filepath.Join(codexHome, "config.toml"), 0600); err != nil {
		return fmt.Errorf("restore %s Codex config: %w", provider, err)
	}

	modelsStatePath := filepath.Join(stateDir, provider+"-models.state")
	stateBytes, err := os.ReadFile(modelsStatePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s model catalogue state: %w", provider, err)
	}
	modelsPath := filepath.Join(codexHome, "models.json")
	if strings.TrimSpace(string(stateBytes)) == "present" {
		if err := copyFile(filepath.Join(stateDir, provider+"-models.json"), modelsPath, 0600); err != nil {
			return fmt.Errorf("restore %s Codex model catalogue: %w", provider, err)
		}
	} else {
		if err := os.Remove(modelsPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove Codex model catalogue: %w", err)
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

func runDeepSeekOfficialInstaller(home, codexHome, action string) error {
	if action != "1" && action != "9" {
		return fmt.Errorf("unsupported DeepSeek setup action %q", action)
	}
	if err := os.MkdirAll(codexHome, 0700); err != nil {
		return fmt.Errorf("create Codex home: %w", err)
	}

	tmp, err := os.CreateTemp("", "setupmjr-deepseek-*.sh")
	if err != nil {
		return fmt.Errorf("create temporary DeepSeek installer: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return err
	}
	defer os.Remove(tmpPath)

	if err := runExternalCommand("curl", "-fsSL", deepSeekCodexSetupURL, "-o", tmpPath); err != nil {
		return fmt.Errorf("download official DeepSeek Codex setup script: %w", err)
	}

	cmd := exec.Command("bash", tmpPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(action + "\n")
	cmd.Env = append(os.Environ(), "CODEX_HOME="+codexHome)
	if action == "1" {
		key, err := deepSeekAPIKey(home, codexHome)
		if err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, "DEEPSEEK_API_KEY="+key)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run official DeepSeek Codex setup script: %w", err)
	}
	return nil
}

func deepSeekAPIKey(home, codexHome string) (string, error) {
	if key := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")); key != "" {
		return key, nil
	}
	for _, path := range []string{
		filepath.Join(codexHome, "config.toml"),
		filepath.Join(home, ".codex", "config.toml"),
		filepath.Join(home, ".codex-deepseek-subagent", "config.toml"),
		filepath.Join(codexStateDir(home), "deepseek-config.toml"),
	} {
		if key := readDeepSeekAPIKey(path); key != "" {
			return key, nil
		}
	}
	fmt.Fprint(os.Stderr, "DeepSeek API key: ")
	keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read DeepSeek API key: %w", err)
	}
	key := strings.TrimSpace(string(keyBytes))
	if key == "" {
		return "", fmt.Errorf("DeepSeek API key cannot be empty")
	}
	return key, nil
}

func readDeepSeekAPIKey(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "experimental_bearer_token") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.TrimSpace(strings.SplitN(parts[1], "#", 2)[0])
		if unquoted, err := strconv.Unquote(value); err == nil {
			return strings.TrimSpace(unquoted)
		}
	}
	return ""
}
