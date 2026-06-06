package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const githubReleasesURL = "https://api.github.com/repos/bloodf/g0router/releases/latest"

// CheckResult reports whether an update is available.
type CheckResult struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	ChangelogURL    string `json:"changelog_url"`
}

// Checker checks for updates against GitHub releases.
type Checker struct {
	client  *http.Client
	url     string
	now     func() time.Time
}

// NewChecker creates a Checker.
func NewChecker() *Checker {
	return &Checker{
		client: &http.Client{Timeout: 15 * time.Second},
		url:    githubReleasesURL,
		now:    time.Now,
	}
}

// Check fetches the latest release and compares it with current.
func (c *Checker) Check(current string) (*CheckResult, error) {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch release: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read release: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := unmarshalJSON(body, &release); err != nil {
		return nil, fmt.Errorf("parse release: %w", err)
	}

	latest := normalizeVersion(release.TagName)
	return &CheckResult{
		Current:         current,
		Latest:          latest,
		UpdateAvailable: isNewer(current, latest),
		ChangelogURL:    release.HTMLURL,
	}, nil
}

// Updater handles downloading and staging updates.
type Updater struct {
	client  *http.Client
	checker *Checker
	baseURL string
}

// NewUpdater creates an Updater.
func NewUpdater() *Updater {
	return &Updater{
		client:  &http.Client{Timeout: 120 * time.Second},
		checker: NewChecker(),
		baseURL: "https://github.com/bloodf/g0router",
	}
}

// Apply downloads the latest binary, verifies its checksum, and stages it.
func (u *Updater) Apply(current, dataDir string) error {
	checker := u.checker
	if checker == nil {
		checker = NewChecker()
	}
	result, err := checker.Check(current)
	if err != nil {
		return fmt.Errorf("check: %w", err)
	}
	if !result.UpdateAvailable {
		return nil
	}

	base := u.baseURL
	if base == "" {
		base = "https://github.com/bloodf/g0router"
	}
	assetName := fmt.Sprintf("g0router-%s-%s", runtime.GOOS, runtime.GOARCH)
	assetURL := fmt.Sprintf("%s/releases/download/v%s/%s", base, result.Latest, assetName)
	checksumsURL := fmt.Sprintf("%s/releases/download/v%s/checksums.txt", base, result.Latest)

	expectedChecksum, err := u.fetchChecksum(checksumsURL, assetName)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	binary, err := u.download(assetURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	sum := sha256.Sum256(binary)
	actualChecksum := hex.EncodeToString(sum[:])
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	stageDir := filepath.Join(dataDir, "update")
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	stagePath := filepath.Join(stageDir, "g0router.new")
	if err := os.WriteFile(stagePath, binary, 0o755); err != nil {
		return fmt.Errorf("write staged file: %w", err)
	}

	return nil
}

func (u *Updater) fetchChecksum(url, assetName string) (string, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", assetName)
}

func (u *Updater) download(url string) ([]byte, error) {
	resp, err := u.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

func isNewer(current, latest string) bool {
	c := normalizeVersion(current)
	l := normalizeVersion(latest)
	return versionGreater(l, c)
}

func versionGreater(a, b string) bool {
	aParts := strings.SplitN(a, "-", 2)
	bParts := strings.SplitN(b, "-", 2)
	aSem := strings.Split(aParts[0], ".")
	bSem := strings.Split(bParts[0], ".")
	for i := 0; i < len(aSem) && i < len(bSem); i++ {
		aNum, _ := strconv.Atoi(aSem[i])
		bNum, _ := strconv.Atoi(bSem[i])
		if aNum != bNum {
			return aNum > bNum
		}
	}
	if len(aSem) != len(bSem) {
		return len(aSem) > len(bSem)
	}
	// If one has a prerelease suffix and the other doesn't, the one without is greater.
	aPre := len(aParts) > 1
	bPre := len(bParts) > 1
	if aPre != bPre {
		return !aPre
	}
	if aPre && bPre {
		return aParts[1] > bParts[1]
	}
	return false
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
