package scheduler

import (
	"context"
	"log"
	"os/exec"
	"strings"
	"time"

	"vasset/media-service/internal/download/config"
)

// YtDLPUpdater 定时检测 yt-dlp 版本更新。
type YtDLPUpdater struct {
	binaryPath string
	enabled    bool
	interval   time.Duration
	timeout    time.Duration
	autoUpdate bool
}

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
	versionCtx, versionCancel := context.WithTimeout(ctx, u.timeout)
	defer versionCancel()

	versionOutput, versionErr := exec.CommandContext(versionCtx, u.binaryPath, "--version").CombinedOutput()
	if versionErr != nil {
		log.Printf("[YtDLPUpdate] Failed to read current version: %v", versionErr)
	} else {
		log.Printf("[YtDLPUpdate] Current version: %s", strings.TrimSpace(string(versionOutput)))
	}

	args := []string{"--update"}
	if !u.autoUpdate {
		args = append(args, "--dry-run")
	}

	updateCtx, updateCancel := context.WithTimeout(ctx, u.timeout)
	defer updateCancel()

	updateOutput, updateErr := exec.CommandContext(updateCtx, u.binaryPath, args...).CombinedOutput()
	if updateErr != nil {
		log.Printf("[YtDLPUpdate] Update check failed: %v output=%s", updateErr, strings.TrimSpace(string(updateOutput)))
		return
	}

	log.Printf("[YtDLPUpdate] Check result: %s", strings.TrimSpace(string(updateOutput)))
}
