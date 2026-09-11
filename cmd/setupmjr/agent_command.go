package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type agentOptions struct {
	codexProvider string
	subagent      string
	showConfig    bool
	list          bool
	help          bool
}

func handleAgent(args []string) error {
	opts, err := parseAgentArgs(args)
	if err != nil {
		return err
	}
	if opts.help || len(args) == 0 {
		printAgentUsage()
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}

	if opts.list {
		printAgentList()
	}
	if opts.codexProvider != "" {
		if err := switchCodexProvider(home, opts.codexProvider); err != nil {
			return err
		}
	}
	if opts.subagent != "" {
		switch opts.subagent {
		case "agy":
			if err := setupSubagentAgy(home); err != nil {
				return err
			}
			fmt.Println("Configured Codex to use agy as the opt-in subagent.")
			if _, err := exec.LookPath("agy"); err != nil {
				fmt.Println("Note: agy is not currently on PATH; install and authenticate Google Antigravity CLI before using the subagent setup.")
			}
		case "deepseek":
			if err := setupSubagentDeepSeek(home); err != nil {
				return err
			}
			fmt.Println("Configured Codex to use DeepSeek Flash as the opt-in subagent.")
		default:
			return fmt.Errorf("unsupported subagent %q", opts.subagent)
		}
	}
	if opts.showConfig {
		if err := printAgentConfig(home); err != nil {
			return err
		}
	}
	return nil
}

func parseAgentArgs(args []string) (agentOptions, error) {
	var opts agentOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			opts.help = true
		case "--list":
			opts.list = true
		case "--config":
			opts.showConfig = true
		case "-c", "--codex-provider":
			value := "openai"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			}
			normalised, err := normaliseCodexProvider(value)
			if err != nil {
				return agentOptions{}, err
			}
			opts.codexProvider = normalised
		case "-s", "--subagent":
			value := "agy"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			}
			normalised, err := normaliseSubagent(value)
			if err != nil {
				return agentOptions{}, err
			}
			opts.subagent = normalised
		case "--subagent-agy":
			fmt.Fprintln(os.Stderr, "Warning: --subagent-agy is deprecated; use --subagent or -s instead.")
			opts.subagent = "agy"
		default:
			return agentOptions{}, fmt.Errorf("unknown agent option: %s", arg)
		}
	}
	return opts, nil
}

func normaliseCodexProvider(value string) (string, error) {
	switch strings.ToLower(value) {
	case "o", "openai", "chatgpt":
		return "openai", nil
	case "d", "deepseek":
		return "deepseek", nil
	default:
		return "", fmt.Errorf("unknown Codex provider %q; use openai (o) or deepseek (d)", value)
	}
}

func normaliseSubagent(value string) (string, error) {
	switch strings.ToLower(value) {
	case "a", "agy":
		return "agy", nil
	case "d", "deepseek":
		return "deepseek", nil
	default:
		return "", fmt.Errorf("unknown subagent %q; use agy or deepseek", value)
	}
}

func printAgentUsage() {
	fmt.Println(`Usage: setupmjr agent [options]

Options:
  -c, --codex-provider [provider]  Switch Codex provider. Defaults to openai when omitted.
  -s, --subagent [subagent]        Configure opt-in subagent. Defaults to agy when omitted.
      --config                     Show the current effective agent configuration.
      --list                       List available providers and subagents.
  -h, --help                       Show this help.

Examples:
  setupmjr agent -c               Switch Codex back to OpenAI/ChatGPT
  setupmjr agent -c d             Switch Codex to DeepSeek
  setupmjr agent -s               Configure agy as the opt-in subagent
  setupmjr agent -s deepseek      Configure DeepSeek as the opt-in subagent`)
}

func printAgentList() {
	fmt.Println(`Codex providers:
  o, openai, chatgpt   OpenAI/ChatGPT (default for -c)
  d, deepseek          DeepSeek

Subagents:
  a, agy               Google Antigravity via agy (default for -s)
  d, deepseek          DeepSeek Flash via isolated Codex`)
}

func printAgentConfig(home string) error {
	provider, err := currentCodexProvider(home)
	if err != nil {
		return err
	}
	subagent, err := currentSubagent(home)
	if err != nil {
		return err
	}
	fmt.Printf("Codex provider: %s\n", provider)
	fmt.Printf("Subagent:       %s\n", subagent)
	return nil
}

func currentSubagent(home string) (string, error) {
	instructions := filepath.Join(home, ".codex", "AGENTS.md")
	content, err := os.ReadFile(instructions)
	if err != nil {
		if os.IsNotExist(err) {
			return "none", nil
		}
		return "", fmt.Errorf("read Codex instructions: %w", err)
	}
	text := string(content)
	hasAgy := strings.Contains(text, "<!-- setupmjr-subagent-agy:start -->")
	hasDeepSeek := strings.Contains(text, "<!-- setupmjr-subagent-deepseek:start -->")
	switch {
	case hasAgy && hasDeepSeek:
		return "multiple (check configuration)", nil
	case hasDeepSeek:
		return "deepseek", nil
	case hasAgy:
		return "agy", nil
	default:
		return "none", nil
	}
}
