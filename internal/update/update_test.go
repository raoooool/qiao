package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func httpClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func tarGzBinary(t *testing.T, name string, contents []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name: name,
		Mode: 0o755,
		Size: int64(len(contents)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(contents); err != nil {
		t.Fatalf("write tar contents: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}

	return buf.Bytes()
}

func TestCheckSkipsForDevVersion(t *testing.T) {
	svc := Service{
		Version:   "dev",
		Repo:      "raoooool/qiao",
		CachePath: filepath.Join(t.TempDir(), "update.yaml"),
		Client: httpClient(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected network call to %s", req.URL)
			return nil, nil
		}),
		Now: time.Now,
	}

	result, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if result.HasUpdate {
		t.Fatalf("expected no update for dev build, got %+v", result)
	}
}

func TestCheckUsesFreshCacheWithoutNetwork(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "update.yaml")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("last_checked_at: 2026-04-04T11:00:00Z\nlatest_version: v1.2.0\n"), 0o600); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	svc := Service{
		Version:   "v1.0.0",
		Repo:      "raoooool/qiao",
		CachePath: cachePath,
		Client: httpClient(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected network call to %s", req.URL)
			return nil, nil
		}),
		Now: func() time.Time {
			return time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
		},
	}

	result, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !result.HasUpdate || result.LatestVersion != "v1.2.0" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckFetchesLatestAndWritesCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "update.yaml")
	svc := Service{
		Version:   "v1.0.0",
		Repo:      "raoooool/qiao",
		CachePath: cachePath,
		Client: httpClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.github.com/repos/raoooool/qiao/releases/latest" {
				t.Fatalf("unexpected URL: %s", req.URL.String())
			}
			return response(http.StatusOK, `{"tag_name":"v1.3.0"}`), nil
		}),
		Now: func() time.Time {
			return time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
		},
	}

	result, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !result.HasUpdate || result.LatestVersion != "v1.3.0" {
		t.Fatalf("unexpected result: %+v", result)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	if !strings.Contains(string(data), "latest_version: v1.3.0") {
		t.Fatalf("expected cache to include latest version, got %q", string(data))
	}
}

func TestUpgradeDownloadsArchiveAndReplacesExecutable(t *testing.T) {
	dir := t.TempDir()
	execPath := filepath.Join(dir, "qiao")
	if err := os.WriteFile(execPath, []byte("old binary"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	archive := tarGzBinary(t, "qiao", []byte("new binary"))
	checksum := sha256Hex(archive)

	svc := Service{
		Version:   "v1.0.0",
		Repo:      "raoooool/qiao",
		CachePath: filepath.Join(dir, "update.yaml"),
		Client: httpClient(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://github.com/raoooool/qiao/releases/download/v1.2.0/qiao_linux_amd64.tar.gz":
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(archive)),
					Header:     make(http.Header),
				}, nil
			case "https://github.com/raoooool/qiao/releases/download/v1.2.0/qiao_checksums.txt":
				return response(http.StatusOK, checksum+"  qiao_linux_amd64.tar.gz\n"), nil
			default:
				t.Fatalf("unexpected URL: %s", req.URL.String())
				return nil, nil
			}
		}),
		Now: func() time.Time { return time.Now().UTC() },
		ExecutablePath: func() (string, error) {
			return execPath, nil
		},
		GOOS:   "linux",
		GOARCH: "amd64",
	}

	result, err := svc.Upgrade(context.Background(), "v1.2.0")
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !result.Updated || result.Version != "v1.2.0" {
		t.Fatalf("unexpected result: %+v", result)
	}

	data, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("read executable: %v", err)
	}
	if got := string(data); got != "new binary" {
		t.Fatalf("expected executable contents to change, got %q", got)
	}
}

func TestUpgradeReturnsAlreadyCurrent(t *testing.T) {
	svc := Service{
		Version:   "v1.2.0",
		Repo:      "raoooool/qiao",
		CachePath: filepath.Join(t.TempDir(), "update.yaml"),
		Client: httpClient(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected network call to %s", req.URL)
			return nil, nil
		}),
		Now: time.Now,
	}

	result, err := svc.Upgrade(context.Background(), "v1.2.0")
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if result.Updated {
		t.Fatalf("expected already-current result, got %+v", result)
	}
}
