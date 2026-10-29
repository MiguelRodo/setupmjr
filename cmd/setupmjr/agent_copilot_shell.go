package main

import (
	"fmt"
	"os"
	"strings"
)

// This non-secret selection is sourced inside a subshell on every invocation.
// Clear stale helpers/tokens/headers as well as endpoints before selecting a route.
const copilotCleanEnvironment = `unset SETUPMJR_COPILOT_ROUTE
unset COPILOT_PROVIDER_TYPE
unset COPILOT_PROVIDER_BASE_URL
unset COPILOT_PROVIDER_API_KEY
unset COPILOT_PROVIDER_API_KEY_COMMAND
unset COPILOT_PROVIDER_BEARER_TOKEN
unset COPILOT_PROVIDER_WIRE_API
unset COPILOT_PROVIDER_TRANSPORT
unset COPILOT_PROVIDER_HEADERS
unset COPILOT_PROVIDER_MODEL_ID
unset COPILOT_PROVIDER_WIRE_MODEL
unset COPILOT_PROVIDER_AZURE_API_VERSION
unset COPILOT_PROVIDER_MAX_PROMPT_TOKENS
unset COPILOT_PROVIDER_MAX_OUTPUT_TOKENS
unset COPILOT_MODEL
unset OPENAI_BASE_URL OPENAI_ORG_ID OPENAI_PROJECT_ID
export COPILOT_PROVIDERS_CONFIG=/dev/null
`

const copilotShellWrapper = `# Managed by setupmjr. Read the latest provider on each plain copilot invocation.
# Preserve user aliases/functions. Reloading this integration is idempotent.
if alias copilot >/dev/null 2>&1 || typeset -f copilot >/dev/null 2>&1; then
    if [ "${SETUPMJR_COPILOT_WRAPPER:-}" != 1 ]; then
        printf 'setupmjr: existing copilot alias/function preserved; plain copilot is not managed.\n' >&2
    fi
else
    function copilot {
        (
            set +x # Never echo expanded credential assignments from a traced parent shell.
            if [ ! -r "$HOME/.config/setupmjr/agent/copilot.env" ]; then
                printf 'setupmjr: Copilot provider selection missing; run setupmjr agent -p github.\n' >&2
                return 1
            fi
            unset SETUPMJR_COPILOT_ROUTE
            . "$HOME/.config/setupmjr/agent/copilot.env" || return 1
            function _setupmjr_copilot_help {
                (
                    unset COPILOT_PROVIDER_API_KEY OPENAI_API_KEY OPENAI_KEY DEEPSEEK_API_KEY
                    command copilot --no-auto-update "$@" 2>/dev/null
                )
            }
            case "${SETUPMJR_COPILOT_ROUTE:-}" in
                openai|deepseek)
                    _setupmjr_help="$(_setupmjr_copilot_help help providers)" || {
                        printf 'setupmjr: Copilot provider help failed; refusing API launch.\n' >&2; return 1;
                    }
                    for _setupmjr_required in COPILOT_PROVIDER_BASE_URL COPILOT_PROVIDER_API_KEY COPILOT_PROVIDER_TYPE; do
                        case "$_setupmjr_help" in
                            *"$_setupmjr_required"*) ;;
                            *) printf 'setupmjr: installed Copilot does not advertise BYOK; refusing API launch.\n' >&2; return 1 ;;
                        esac
                    done
                    if [ "$SETUPMJR_COPILOT_ROUTE" = openai ]; then
                        for _setupmjr_required in COPILOT_PROVIDER_WIRE_API responses; do
                            case "$_setupmjr_help" in
                                *"$_setupmjr_required"*) ;;
                                *) printf 'setupmjr: installed Copilot lacks Responses support; refusing API launch.\n' >&2; return 1 ;;
                            esac
                        done
                        _setupmjr_help="$(_setupmjr_copilot_help --help)" || return 1
                        for _setupmjr_required in --model --reasoning-effort --no-auto-update; do
                            case "$_setupmjr_help" in
                                *"$_setupmjr_required"*) ;;
                                *) printf 'setupmjr: installed Copilot lacks required launch flags; refusing API launch.\n' >&2; return 1 ;;
                            esac
                        done
                    fi
                    ;;
            esac
            unset OPENAI_API_KEY OPENAI_KEY DEEPSEEK_API_KEY
            case "${SETUPMJR_COPILOT_ROUTE:-}" in
                openai) command copilot --no-auto-update --model gpt-6.1-sol --reasoning-effort medium "$@" ;;
                github|deepseek) command copilot --no-auto-update "$@" ;;
                *) printf 'setupmjr: invalid Copilot provider selection; refusing to launch.\n' >&2; return 1 ;;
            esac
        )
    }
    SETUPMJR_COPILOT_WRAPPER=1
fi
`

func configureCopilotOpenAI(home string) error {
	content := copilotCleanEnvironment + `# Managed by setupmjr. OpenAI API is the active Copilot model provider.
SETUPMJR_COPILOT_ROUTE=openai
export COPILOT_PROVIDER_TYPE='openai'
export COPILOT_PROVIDER_BASE_URL='https://api.openai.com/v1'
export COPILOT_PROVIDER_WIRE_API='responses'
export COPILOT_MODEL='gpt-6.1-sol'
export COPILOT_PROVIDER_MAX_PROMPT_TOKENS='272000'
export COPILOT_PROVIDER_MAX_OUTPUT_TOKENS='128000'
case "${OPENAI_API_KEY:-}" in
    *[![:space:]]*) export COPILOT_PROVIDER_API_KEY="$OPENAI_API_KEY" ;;
    *) case "${OPENAI_KEY:-}" in
        *[![:space:]]*) export COPILOT_PROVIDER_API_KEY="$OPENAI_KEY" ;;
        *) printf 'OpenAI key missing; export OPENAI_API_KEY or OPENAI_KEY in this terminal. No fallback to Copilot allowances.\n' >&2; return 1 ;;
    esac ;;
esac
`
	if err := writeCopilotEnv(home, content); err != nil {
		return err
	}
	if err := wireCopilotEnv(home); err != nil {
		return err
	}
	fmt.Println("Copilot provider selected: OpenAI API, Responses, Sol 6.1 (medium). No credential was stored.")
	printCopilotShellInstructions()
	return nil
}

func printCopilotShellInstructions() {
	fmt.Println("First use in an existing Bash/Zsh terminal: source ~/.config/setupmjr/agent/copilot.sh")
	fmt.Println("Then run copilot normally. Later provider switches apply to the next copilot invocation.")
	fmt.Println("Configured route/status: setupmjr agent copilot-status (actual API billing is not verified).")
}

func copilotKeyPresent(home, provider string) bool {
	switch provider {
	case "openai":
		return strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) != "" || strings.TrimSpace(os.Getenv("OPENAI_KEY")) != ""
	case "deepseek":
		if strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY")) != "" {
			return true
		}
		info, err := os.Stat(deepSeekCredentialPath(home))
		return err == nil && info.Mode().IsRegular() && info.Size() > 0
	default:
		return false
	}
}

func printCopilotStatus(home string) {
	provider := currentCopilotProvider(home)
	fmt.Printf("Copilot provider: %s (configured route)\n", provider)
	switch provider {
	case "openai":
		fmt.Println("Endpoint: https://api.openai.com/v1; wire API: responses; model: gpt-6.1-sol; reasoning: medium")
	case "deepseek":
		fmt.Printf("Endpoint: %s; wire API: anthropic; model: %s\n", copilotDeepSeekBaseURL, copilotDeepSeekModel)
	case "github":
		fmt.Println("Endpoint/wire API: GitHub-managed; model: Copilot default or your explicit selection; route: Copilot allowances")
	default:
		fmt.Println("Custom configuration; inspect provider settings locally.")
	}
	if provider == "openai" || provider == "deepseek" {
		if copilotKeyPresent(home, provider) {
			fmt.Println("API key: present in this environment or managed credential source (not validated)")
		} else {
			fmt.Println("API key: missing in this environment; launch will fail without fallback")
		}
	}
	fmt.Println("Plain copilot requires the managed Bash/Zsh function; run type copilot to check resolution.")
	fmt.Println("This reports configured routing, not a verified request or billing/grant ledger entry.")
}
