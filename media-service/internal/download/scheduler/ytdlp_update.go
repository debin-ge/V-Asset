package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"youdlp/media-service/internal/download/config"
)

// YtDLPUpdater 定时检测 yt-dlp 版本更新。
type YtDLPUpdater struct {
	binaryPath string
	enabled    bool
	interval   time.Duration
	timeout    time.Duration
	autoUpdate bool
}

const latestReleaseAPI = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"

func NewYtDLPUpdater(ytdlpCfg *config.YtDLPConfig, updateCfg *config.YtDLPUpdateConfig) *YtDLPUpdater {
	return &YtDLPUpdater{
		binaryPath: ytdlpCfg.BinaryPath,
		enabled:    updateCfg.Enabled,
		interval:   time.Duration(updateCfg.IntervalHours) * time.Hour,
		timeout:    time.Duration(updateCfg.TimeoutSeconds) * time.Second,
		autoUpdate: updateCfg.AutoUpdate,
	}
}

func (u *YtDLPUpdater) Start(ctx context.Context) {
	if !u.enabled {
		log.Println("[YtDLPUpdate] Scheduler is disabled")
		return
	}

	if u.interval <= 0 {
		u.interval = 6 * time.Hour
	}
	if u.timeout <= 0 {
		u.timeout = 30 * time.Second
	}

	log.Printf("[YtDLPUpdate] Starting scheduler, interval=%v auto_update=%v", u.interval, u.autoUpdate)
	u.checkOnce(ctx)

	ticker := time.NewTicker(u.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.checkOnce(ctx)
		case <-ctx.Done():
			log.Println("[YtDLPUpdate] Scheduler stopped")
			return
		}
	}
}

func (u *YtDLPUpdater) checkOnce(ctx context.Context) {
	currentVersion, versionErr := u.getCurrentVersion(ctx)
	if versionErr != nil {
		log.Printf("[YtDLPUpdate] Failed to read current version: %v", versionErr)
	} else {
		log.Printf("[YtDLPUpdate] Current version: %s", currentVersion)
	}

	if !u.autoUpdate {
		u.logLatestReleaseCheck(ctx, currentVersion)
		return
	}

	args := []string{"--update"}

	updateCtx, updateCancel := context.WithTimeout(ctx, u.timeout)
	defer updateCancel()

	updateOutput, updateErr := exec.CommandContext(updateCtx, u.binaryPath, args...).CombinedOutput()
	if updateErr != nil {
		log.Printf("[YtDLPUpdate] Update check failed: %v output=%s", updateErr, strings.TrimSpace(string(updateOutput)))
		return
	}

	log.Printf("[YtDLPUpdate] Check result: %s", strings.TrimSpace(string(updateOutput)))
}

func (u *YtDLPUpdater) getCurrentVersion(ctx context.Context) (string, error) {
	versionCtx, versionCancel := context.WithTimeout(ctx, u.timeout)
	defer versionCancel()

	versionOutput, versionErr := exec.CommandContext(versionCtx, u.binaryPath, "--version").CombinedOutput()
	if versionErr != nil {
		return "", versionErr
	}

	return normalizeVersion(strings.TrimSpace(string(versionOutput))), nil
}

func (u *YtDLPUpdater) logLatestReleaseCheck(ctx context.Context, currentVersion string) {
	latestVersion, err := u.getLatestReleaseVersion(ctx)
	if err != nil {
		log.Printf("[YtDLPUpdate] Failed to fetch latest release: %v", err)
		return
	}

	if currentVersion == "" {
		log.Printf("[YtDLPUpdate] Latest release: %s", latestVersion)
		return
	}

	switch compareVersions(currentVersion, latestVersion) {
	case -1:
		log.Printf("[YtDLPUpdate] Update available: current=%s latest=%s", currentVersion, latestVersion)
	case 1:
		log.Printf("[YtDLPUpdate] Current version %s is newer than latest release %s", currentVersion, latestVersion)
	default:
		log.Printf("[YtDLPUpdate] Already up to date: %s", currentVersion)
	}
}

func (u *YtDLPUpdater) getLatestReleaseVersion(ctx context.Context) (string, error) {
	requestCtx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "youdlp-media-service")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	if payload.TagName == "" {
		return "", fmt.Errorf("tag_name is empty")
	}

	return normalizeVersion(payload.TagName), nil
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if strings.Contains(version, "@") {
		version = version[strings.LastIndex(version, "@")+1:]
	}
	version = strings.TrimPrefix(version, "v")
	return version
}

func compareVersions(current, latest string) int {
	currentParts := versionParts(current)
	latestParts := versionParts(latest)
	maxLen := len(currentParts)
	if len(latestParts) > maxLen {
		maxLen = len(latestParts)
	}

	for i := 0; i < maxLen; i++ {
		currentPart := 0
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}
		latestPart := 0
		if i < len(latestParts) {
			latestPart = latestParts[i]
		}

		switch {
		case currentPart < latestPart:
			return -1
		case currentPart > latestPart:
			return 1
		}
	}

	return 0
}

func versionParts(version string) []int {
	fields := strings.FieldsFunc(version, func(r rune) bool {
		return r < '0' || r > '9'
	})

	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		value, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		parts = append(parts, value)
	}
	return parts
}
