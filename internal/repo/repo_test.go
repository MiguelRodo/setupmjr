package repo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSetupRepoDevcontainerTreatsRepoAndBranchAsData(t *testing.T) {
	tests := []struct {
		name   string
		repo   string
		branch string
	}{
		{name: "normal", repo: "owner/comp", branch: "main"},
		{name: "repo shell metacharacters", repo: "owner/comp$(touch${IFS}injected)", branch: "main"},
		{name: "branch shell metacharacters", repo: "owner/comp", branch: "main$(touch${IFS}injected)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			repoName := strings.SplitN(tt.repo, "/", 2)[1]

			originalTransport := http.DefaultClient.Transport
			http.DefaultClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "github.com" {
					t.Fatalf("unexpected download host: %s", req.URL.Host)
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     "200 OK",
					Header:     make(http.Header),
					Body:       io.NopCloser(devcontainerArchive(t, repoName+"-"+tt.branch)),
					Request:    req,
				}, nil
			})
			t.Cleanup(func() { http.DefaultClient.Transport = originalTransport })

			err := SetupRepoDevcontainer(tt.repo, tt.branch, false)
			if _, statErr := os.Stat("injected"); !os.IsNotExist(statErr) {
				t.Fatalf("shell syntax was executed; injected file stat error = %v", statErr)
			}
			if err != nil {
				t.Fatalf("SetupRepoDevcontainer() error = %v", err)
			}
			if _, err := os.Stat(filepath.Join(".devcontainer", "devcontainer.json")); err != nil {
				t.Fatalf("expected extracted devcontainer: %v", err)
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
