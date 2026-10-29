package repo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

type readmeContact struct{ name, email string }
type readmeAnswers struct {
	title, purpose string
	contacts       [2]readmeContact
}

func SetupRepoReadme() error {
	return setupRepoReadme(os.Stdin, os.Stdout)
}

func setupRepoReadme(in io.Reader, out io.Writer) error {
	if _, err := os.Lstat("README.md"); err == nil {
		return fmt.Errorf("README.md already exists; refusing to overwrite existing content")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check README.md: %w", err)
	}
	answers, err := readReadmeAnswers(in, out)
	if err != nil {
		return err
	}
	// Exclusive creation also protects a README created while prompts were open.
	file, err := os.OpenFile("README.md", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("create README.md: %w", err)
	}
	_, writeErr := io.WriteString(file, renderReadme(answers))
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = os.Remove("README.md") // Only the file this invocation exclusively created.
		return fmt.Errorf("write README.md: %w", err)
	}
	_, err = fmt.Fprintln(out, "Successfully created README.md")
	return err
}

func readReadmeAnswers(in io.Reader, out io.Writer) (answers readmeAnswers, err error) {
	reader := bufio.NewReader(in)
	readLine := func(prompt string) (string, error) {
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return "", err
		}
		line, err := reader.ReadString('\n')
		if err == io.EOF && len(line) > 0 {
			err = nil // A final piped answer need not end with a newline.
		}
		return strings.TrimSpace(line), err
	}
	input, inputFile := in.(*os.File)
	output, outputFile := out.(*os.File)
	// Native Windows keeps its console's line-input behavior; the POSIX editor
	// explicitly controls VT100 redraws so wrapped deletion stays visible.
	if runtime.GOOS != "windows" && inputFile && outputFile && term.IsTerminal(int(input.Fd())) && term.IsTerminal(int(output.Fd())) {
		width, height, sizeErr := term.GetSize(int(output.Fd()))
		if sizeErr != nil {
			return answers, fmt.Errorf("read terminal dimensions: %w", sizeErr)
		}
		previous, rawErr := term.MakeRaw(int(input.Fd()))
		if rawErr != nil {
			return answers, fmt.Errorf("start terminal line editing: %w", rawErr)
		}
		defer func() {
			if restoreErr := term.Restore(int(input.Fd()), previous); restoreErr != nil {
				err = errors.Join(err, fmt.Errorf("restore terminal: %w", restoreErr))
			}
		}()
		editor := term.NewTerminal(struct {
			io.Reader
			io.Writer
		}{input, output}, "")
		if err := editor.SetSize(width, height); err != nil {
			return answers, fmt.Errorf("set terminal dimensions: %w", err)
		}
		readLine = func(prompt string) (string, error) {
			editor.SetPrompt(prompt)
			line, err := editor.ReadLine()
			if err != nil {
				_, _ = fmt.Fprint(out, "\r\n")
			}
			return strings.TrimSpace(line), err
		}
	}
	for _, question := range []struct {
		prompt string
		answer *string
	}{
		{"Enter title: ", &answers.title},
		{"Complete \"This repository ...\": ", &answers.purpose},
		{"Contact 1 name: ", &answers.contacts[0].name},
		{"Contact 1 email: ", &answers.contacts[0].email},
		{"Contact 2 name (optional): ", &answers.contacts[1].name},
		{"Contact 2 email (optional): ", &answers.contacts[1].email},
	} {
		answer, err := readLine(question.prompt)
		if err != nil {
			return answers, fmt.Errorf("read README answer (%s): %w", strings.TrimSpace(question.prompt), err)
		}
		*question.answer = answer
	}
	return answers, nil
}

func renderReadme(answers readmeAnswers) string {
	var body strings.Builder
	fmt.Fprintf(&body, "# %s\n\nThis repository %s\n", answers.title, answers.purpose)
	var contacts []string
	for _, contact := range answers.contacts {
		switch {
		case contact.name != "" && contact.email != "":
			contacts = append(contacts, fmt.Sprintf("- %s: %s", contact.name, contact.email))
		case contact.name != "":
			contacts = append(contacts, "- "+contact.name)
		case contact.email != "":
			contacts = append(contacts, "- "+contact.email)
		}
	}
	if len(contacts) > 0 {
		fmt.Fprintf(&body, "\n## Contact\n\nFor more information, please contact:\n\n%s\n", strings.Join(contacts, "\n"))
	}
	body.WriteString(`
## Links

- [URLs to data sources (e.g. OneDrive), GitHub repositories, publications, etc.]

## Details

[Methods, timeline, team, data sources, software/tools, etc.]
`)
	return body.String()
}
