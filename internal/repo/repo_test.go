package repo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractDevcontainerTreatsArchiveMemberAsData(t *testing.T) {
	tests := []struct {
		name        string
		repoDirName string
	}{
		{name: "normal", repoDirName: "comp-main"},
		{name: "shell metacharacters", repoDirName: "comp-main$(touch injected)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())

			output, err := extractDevcontainer(devcontainerArchive(t, tt.repoDirName), tt.repoDirName)
			if err != nil {
				t.Fatalf("extractDevcontainer() error = %v, output = %s", err, output)
			}
			if _, err := os.Stat(filepath.Join(".devcontainer", "devcontainer.json")); err != nil {
				t.Fatalf("expected extracted devcontainer: %v", err)
			}
			if _, err := os.Stat("injected"); !os.IsNotExist(err) {
				t.Fatalf("shell syntax was executed; injected file stat error = %v", err)
			}
		})
	}
}

func devcontainerArchive(t *testing.T, repoDirName string) *bytes.Reader {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	dir := repoDirName + "/.devcontainer"
	if err := tw.WriteHeader(&tar.Header{Name: dir, Mode: 0755, Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}
	body := []byte("{}\n")
	if err := tw.WriteHeader(&tar.Header{Name: dir + "/devcontainer.json", Mode: 0644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	return bytes.NewReader(buf.Bytes())
}
