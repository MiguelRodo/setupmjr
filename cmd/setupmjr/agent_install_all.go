package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
)

func installCodingAgents() error {
	return installCodingAgentsForOS(runtime.GOOS, os.Stdout)
}

func installCodingAgentsForOS(goos string, out io.Writer) error {
	var failures []error
	for _, agent := range []string{"claude", "codex", "copilot"} {
		if path, err := codingAgentLookPath(agent); err == nil {
			fmt.Fprintf(out, "%s: already installed at %s; skipped.\n", agent, path)
			continue
		}
		if err := installCodingAgentForOS(agent, goos); err != nil {
			fmt.Fprintf(out, "%s: failed: %v\n", agent, err)
			failures = append(failures, err)
			continue
		}
		fmt.Fprintf(out, "%s: installer completed. Restart your shell if needed; sign-in is separate.\n", agent)
	}
	return errors.Join(failures...)
}
