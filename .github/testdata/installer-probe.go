package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	got := strings.Join(os.Args[1:], " ")
	if got != "install --repos" {
		fmt.Fprintf(os.Stderr, "unexpected setupmjr installer call: %q\n", got)
		os.Exit(1)
	}
	if path := os.Getenv("SETUPMJR_INSTALLER_CALL_LOG"); path != "" {
		if err := os.WriteFile(path, []byte(got+"\n"), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
