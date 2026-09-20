package netutil

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	originalClient := Client
	Client = &http.Client{Timeout: 20 * time.Millisecond}
	defer func() { Client = originalClient }()

	_, err := Get(server.URL)
	if err == nil {
		t.Fatal("timed out request unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), server.URL) {
		t.Fatalf("timeout error %q does not contain request URL", err)
	}
}

func TestGetPreservesHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	defer server.Close()

	_, err := Get(server.URL)
	if err == nil {
		t.Fatal("403 request unexpectedly succeeded")
	}
	for _, want := range []string{server.URL, "403 Forbidden"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}
}

func TestDownloadFilePreservesDestinationOnPartialResponse(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "workflow.yml")
	if err := os.WriteFile(dest, []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("partial"))
	}))
	defer server.Close()

	if err := DownloadFile(server.URL, dest); err == nil {
		t.Fatal("partial response unexpectedly succeeded")
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing\n" {
		t.Fatalf("destination changed after failed download: %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "workflow.yml" {
		t.Fatalf("failed download left temporary files: %#v", entries)
	}
}

func TestDownloadFileReplacesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "workflow.yml")
	if err := os.WriteFile(dest, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("new\n"))
	}))
	defer server.Close()

	if err := DownloadFile(server.URL, dest); err != nil {
		t.Fatalf("DownloadFile() error = %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Fatalf("destination = %q, want new content", got)
	}
}
