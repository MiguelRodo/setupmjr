package netutil

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const RequestTimeout = 60 * time.Second

var Client = &http.Client{Timeout: RequestTimeout}

func Get(url string) (*http.Response, error) {
	resp, err := Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	if resp.StatusCode != http.StatusOK {
		status := resp.Status
		_ = resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, status)
	}
	return resp, nil
}

func DownloadFile(url, dest string) error {
	resp, err := Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary download for %s: %w", dest, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("download %s: %w", url, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary download for %s: %w", dest, err)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return fmt.Errorf("set download permissions for %s: %w", dest, err)
	}
	if err := os.Rename(tmpPath, dest); err != nil {
		return fmt.Errorf("replace %s after download: %w", dest, err)
	}
	return nil
}
