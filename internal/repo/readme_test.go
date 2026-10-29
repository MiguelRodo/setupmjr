package repo

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestReadmeOptionalContactsAndFinalNewline(t *testing.T) {
	for _, test := range []struct {
		name     string
		contacts string
		want     string
	}{
		{"one contact", "Alex Example\nalex@example.test\n  \n \n", "- Alex Example: alex@example.test\n"},
		{"two contacts", "Alex Example\nalex@example.test\nSam Example\nsam@example.test", "- Alex Example: alex@example.test\n- Sam Example: sam@example.test\n"},
		{"name only", "Alex Example\n\n\n\n", "- Alex Example\n"},
		{"email only", "\nalex@example.test\n\n\n", "- alex@example.test\n"},
		{"no contacts", "\n\n\n\n", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			input := "Example\n compares tools. \n" + test.contacts
			if err := setupRepoReadme(strings.NewReader(input), io.Discard); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile("README.md")
			if err != nil {
				t.Fatal(err)
			}
			text := string(content)
			if !strings.HasPrefix(text, "# Example\n\nThis repository compares tools.\n") || !strings.HasSuffix(text, "\n") || strings.HasSuffix(text, "\n\n") {
				t.Fatal("title/purpose normalization or final newline incorrect")
			}
			if test.want == "" {
				if strings.Contains(text, "## Contact") {
					t.Fatal("empty contact section was emitted")
				}
			} else if !strings.Contains(text, "please contact:\n\n"+test.want+"\n## Links") {
				t.Fatalf("incorrect contact formatting:\n%s", text)
			}
			if strings.Contains(text, "\n  -") || strings.Contains(text, "- :") || strings.Contains(text, "- \n") {
				t.Fatal("empty or nested contact bullet")
			}
		})
	}
}

func TestReadmePreservesExistingFileAndAbortsIncompleteInput(t *testing.T) {
	t.Run("existing README", func(t *testing.T) {
		t.Chdir(t.TempDir())
		original := []byte("# User edits\nKeep these.\n")
		if err := os.WriteFile("README.md", original, 0644); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := setupRepoReadme(strings.NewReader(""), &output); err == nil {
			t.Fatal("existing README was accepted")
		}
		got, err := os.ReadFile("README.md")
		if err != nil || !bytes.Equal(got, original) || output.Len() != 0 {
			t.Fatal("existing README changed or questionnaire started")
		}
	})
	t.Run("existing symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation can require privileges on Windows")
		}
		t.Chdir(t.TempDir())
		if err := os.WriteFile("user-readme", []byte("Keep this.\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("user-readme", "README.md"); err != nil {
			t.Fatal(err)
		}
		if err := setupRepoReadme(strings.NewReader(""), io.Discard); err == nil {
			t.Fatal("existing README symlink was accepted")
		}
		got, err := os.ReadFile("user-readme")
		if err != nil || string(got) != "Keep this.\n" {
			t.Fatal("modified the existing symlink target")
		}
	})
	t.Run("incomplete input", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := setupRepoReadme(strings.NewReader("Example\n"), io.Discard); err == nil {
			t.Fatal("unexpected EOF was ignored")
		}
		if _, err := os.Stat("README.md"); !os.IsNotExist(err) {
			t.Fatal("incomplete answers created a README")
		}
	})
}

type readmeWriterFunc func([]byte) (int, error)

func (f readmeWriterFunc) Write(p []byte) (int, error) { return f(p) }

func TestReadmePreservesFileCreatedDuringPrompts(t *testing.T) {
	t.Chdir(t.TempDir())
	created := false
	output := readmeWriterFunc(func(p []byte) (int, error) {
		if !created {
			created = true
			if err := os.WriteFile("README.md", []byte("Concurrent user edits.\n"), 0644); err != nil {
				return 0, err
			}
		}
		return len(p), nil
	})
	if err := setupRepoReadme(strings.NewReader("Example\ncompares tools.\n\n\n\n\n"), output); err == nil {
		t.Fatal("overwrote a README created while prompts were open")
	}
	got, err := os.ReadFile("README.md")
	if err != nil || string(got) != "Concurrent user edits.\n" {
		t.Fatal("concurrent README edits were lost")
	}
}
