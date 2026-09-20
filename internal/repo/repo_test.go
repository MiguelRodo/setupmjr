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

	"github.com/MiguelRodo/setupmjr/internal/netutil"
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

			originalTransport := netutil.Client.Transport
			netutil.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
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
			t.Cleanup(func() { netutil.Client.Transport = originalTransport })

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

func TestSetupRepoActionFetchesLatestReleasedExample(t *testing.T) {
	t.Chdir(t.TempDir())
	originalTransport := netutil.Client.Transport
	defer func() { netutil.Client.Transport = originalTransport }()

	var requests []string
	netutil.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.URL.String())
		var body string
		switch {
		case req.URL.Host == "api.github.com" && req.URL.Path == "/repos/MiguelRodo/actions/releases/latest":
			body = `{"tag_name":"v9.8.7"}`
		case req.URL.Host == "raw.githubusercontent.com" && req.URL.Path == "/MiguelRodo/actions/v9.8.7/examples/version-release.yml":
			body = "name: canonical release example\n"
		default:
			t.Fatalf("unexpected request: %s", req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})

	if err := SetupRepoAction("version-release"); err != nil {
		t.Fatalf("SetupRepoAction() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(".github", "workflows", "version-release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "name: canonical release example\n" {
		t.Fatalf("workflow = %q", got)
	}
	if len(requests) != 2 {
		t.Fatalf("requests = %#v, want release lookup plus workflow download", requests)
	}
	for _, request := range requests {
		if strings.Contains(request, "/main/") {
			t.Fatalf("workflow fetch used mutable main: %s", request)
		}
	}
}

func TestSetupRepoActionReportsMissingReleasedExample(t *testing.T) {
	t.Chdir(t.TempDir())
	originalTransport := netutil.Client.Transport
	defer func() { netutil.Client.Transport = originalTransport }()

	netutil.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		status := http.StatusNotFound
		statusLine := "404 Not Found"
		body := ""
		if req.URL.Host == "api.github.com" {
			status = http.StatusOK
			statusLine = "200 OK"
			body = `{"tag_name":"v9.8.7"}`
		}
		return &http.Response{
			StatusCode: status,
			Status:     statusLine,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})

	err := SetupRepoAction("missing-action")
	if err == nil {
		t.Fatal("missing released workflow unexpectedly succeeded")
	}
	for _, want := range []string{"missing-action", "v9.8.7", "404 Not Found"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
	if _, statErr := os.Stat(filepath.Join(".github", "workflows", "missing-action.yml")); !os.IsNotExist(statErr) {
		t.Fatalf("missing workflow left a destination file: %v", statErr)
	}
}

func TestSetupRepoActionReportsReleaseLookupFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	originalTransport := netutil.Client.Transport
	defer func() { netutil.Client.Transport = originalTransport }()

	netutil.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.github.com" {
			t.Fatalf("workflow download attempted after release lookup failure: %s", req.URL)
		}
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Status:     "403 Forbidden",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    req,
		}, nil
	})

	err := SetupRepoAction("version-release")
	if err == nil {
		t.Fatal("release lookup failure unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "failed to fetch latest actions release") || !strings.Contains(err.Error(), "403 Forbidden") {
		t.Fatalf("unexpected error: %v", err)
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
