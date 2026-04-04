package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultRepo = "raoooool/qiao"

type Service struct {
	Version        string
	Repo           string
	CachePath      string
	Client         *http.Client
	Now            func() time.Time
	ExecutablePath func() (string, error)
	GOOS           string
	GOARCH         string
}

type CheckResult struct {
	LatestVersion string
	HasUpdate     bool
}

type UpgradeResult struct {
	Version string
	Updated bool
}

type release struct {
	TagName string `json:"tag_name"`
}

type cacheFile struct {
	LastCheckedAt time.Time `yaml:"last_checked_at"`
	LatestVersion string    `yaml:"latest_version"`
}

func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "qiao", "update.yaml"), nil
}

func (s Service) Check(ctx context.Context) (CheckResult, error) {
	s = s.withDefaults()
	if s.Version == "dev" {
		return CheckResult{}, nil
	}

	cache, fresh := s.loadFreshCache()
	if fresh {
		return checkResultFromVersion(s.Version, cache.LatestVersion)
	}

	latest, err := s.latestRelease(ctx)
	if err != nil {
		return CheckResult{}, err
	}

	_ = s.saveCache(cacheFile{
		LastCheckedAt: s.Now().UTC(),
		LatestVersion: latest,
	})

	return checkResultFromVersion(s.Version, latest)
}

func (s Service) Upgrade(ctx context.Context, targetVersion string) (UpgradeResult, error) {
	s = s.withDefaults()

	version := targetVersion
	if version == "" {
		latest, err := s.latestRelease(ctx)
		if err != nil {
			return UpgradeResult{}, err
		}
		version = latest
	}

	if s.Version != "dev" && version == s.Version {
		return UpgradeResult{Version: version, Updated: false}, nil
	}

	execPath, err := s.ExecutablePath()
	if err != nil {
		return UpgradeResult{}, fmt.Errorf("resolve executable path: %w", err)
	}

	archiveName, err := releaseArchiveName(s.GOOS, s.GOARCH)
	if err != nil {
		return UpgradeResult{}, err
	}

	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", s.Repo, version)
	archiveData, err := s.download(ctx, baseURL+"/"+archiveName)
	if err != nil {
		return UpgradeResult{}, err
	}
	checksumData, err := s.download(ctx, baseURL+"/qiao_checksums.txt")
	if err != nil {
		return UpgradeResult{}, err
	}

	expectedChecksum, err := checksumForAsset(string(checksumData), archiveName)
	if err != nil {
		return UpgradeResult{}, err
	}
	if got := sha256Hex(archiveData); got != expectedChecksum {
		return UpgradeResult{}, errors.New("checksum mismatch")
	}

	binary, err := extractBinary(archiveData, s.GOOS)
	if err != nil {
		return UpgradeResult{}, err
	}

	if err := replaceExecutable(execPath, binary, s.GOOS); err != nil {
		return UpgradeResult{}, err
	}

	return UpgradeResult{Version: version, Updated: true}, nil
}

func (s Service) withDefaults() Service {
	if s.Repo == "" {
		s.Repo = defaultRepo
	}
	if s.CachePath == "" {
		cachePath, _ := DefaultCachePath()
		s.CachePath = cachePath
	}
	if s.Client == nil {
		s.Client = &http.Client{Timeout: 15 * time.Second}
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.ExecutablePath == nil {
		s.ExecutablePath = os.Executable
	}
	if s.GOOS == "" {
		s.GOOS = runtime.GOOS
	}
	if s.GOARCH == "" {
		s.GOARCH = runtime.GOARCH
	}
	return s
}

func (s Service) loadFreshCache() (cacheFile, bool) {
	data, err := os.ReadFile(s.CachePath)
	if err != nil {
		return cacheFile{}, false
	}

	var cache cacheFile
	if err := yaml.Unmarshal(data, &cache); err != nil {
		return cacheFile{}, false
	}
	if cache.LastCheckedAt.IsZero() {
		return cache, false
	}
	if s.Now().UTC().Sub(cache.LastCheckedAt.UTC()) >= 24*time.Hour {
		return cache, false
	}

	return cache, true
}

func (s Service) saveCache(cache cacheFile) error {
	data, err := yaml.Marshal(cache)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.CachePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.CachePath, data, 0o600)
}

func (s Service) latestRelease(ctx context.Context) (string, error) {
	data, err := s.download(ctx, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", s.Repo))
	if err != nil {
		return "", err
	}

	var rel release
	if err := json.Unmarshal(data, &rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", errors.New("latest release tag is empty")
	}
	return rel.TagName, nil
}

func (s Service) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func checkResultFromVersion(current, latest string) (CheckResult, error) {
	if latest == "" {
		return CheckResult{}, nil
	}
	hasUpdate, err := isNewerVersion(current, latest)
	if err != nil {
		return CheckResult{}, nil
	}
	return CheckResult{
		LatestVersion: latest,
		HasUpdate:     hasUpdate,
	}, nil
}

func isNewerVersion(current, latest string) (bool, error) {
	currentVersion, err := parseVersion(current)
	if err != nil {
		return false, err
	}
	latestVersion, err := parseVersion(latest)
	if err != nil {
		return false, err
	}

	if latestVersion.major != currentVersion.major {
		return latestVersion.major > currentVersion.major, nil
	}
	if latestVersion.minor != currentVersion.minor {
		return latestVersion.minor > currentVersion.minor, nil
	}
	return latestVersion.patch > currentVersion.patch, nil
}

type semVersion struct {
	major int
	minor int
	patch int
}

func parseVersion(v string) (semVersion, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return semVersion{}, fmt.Errorf("invalid version %q", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semVersion{}, fmt.Errorf("invalid version %q", v)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semVersion{}, fmt.Errorf("invalid version %q", v)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semVersion{}, fmt.Errorf("invalid version %q", v)
	}

	return semVersion{major: major, minor: minor, patch: patch}, nil
}

func releaseArchiveName(goos, goarch string) (string, error) {
	switch goos {
	case "linux", "darwin":
		return fmt.Sprintf("qiao_%s_%s.tar.gz", goos, goarch), nil
	case "windows":
		return fmt.Sprintf("qiao_%s_%s.zip", goos, goarch), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", goos)
	}
}

func checksumForAsset(checksums, asset string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == asset {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum entry not found for %s", asset)
}

func extractBinary(archive []byte, goos string) ([]byte, error) {
	binary := "qiao"
	if goos == "windows" {
		binary = "qiao.exe"
		return extractZipBinary(archive, binary)
	}
	return extractTarGzBinary(archive, binary)
}

func extractTarGzBinary(archive []byte, binary string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) != binary {
			continue
		}
		return io.ReadAll(tr)
	}
	return nil, fmt.Errorf("could not find %s binary in archive", binary)
}

func extractZipBinary(archive []byte, binary string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, err
	}

	for _, file := range reader.File {
		if filepath.Base(file.Name) != binary {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		return io.ReadAll(rc)
	}
	return nil, fmt.Errorf("could not find %s binary in archive", binary)
}

func replaceExecutable(execPath string, binary []byte, goos string) error {
	dir := filepath.Dir(execPath)
	tempFile, err := os.CreateTemp(dir, filepath.Base(execPath)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := tempFile.Name()

	cleanup := func() {
		_ = tempFile.Close()
		_ = os.Remove(tempName)
	}

	if _, err := tempFile.Write(binary); err != nil {
		cleanup()
		return err
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempName)
		return err
	}
	if goos != "windows" {
		if err := os.Chmod(tempName, 0o755); err != nil {
			_ = os.Remove(tempName)
			return err
		}
	}
	if err := os.Rename(tempName, execPath); err != nil {
		_ = os.Remove(tempName)
		return err
	}

	return nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
