package worker

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"youdlp/media-service/internal/download/config"
	"youdlp/media-service/internal/download/models"
	"youdlp/media-service/internal/download/repository"
	"youdlp/media-service/internal/download/storage"
	"youdlp/media-service/internal/download/ytdlp"
	"youdlp/media-service/internal/platformpolicy"
	"youdlp/media-service/internal/redact"
)

// AssetClientInterface Asset 服务客户端接口
type AssetClientInterface interface {
	GetCookieContent(cookieID int64, platform, taskID string) (string, error)
	ReportCookieUsage(cookieID int64, success bool, taskID string) error
	ReportProxyUsage(taskID, proxyLeaseID, stage string, success bool) error
	CleanupCookieFile(cookieFile string) error
	UpdateHistoryCompleted(taskID, filePath, fileName, fileHash string, fileSize int64, pendingCleanup bool) error
	UpdateHistoryFailed(taskID, errorMessage string) error
	CaptureIngressUsage(taskID string, actualIngressBytes int64) error
	ReleaseInitialDownload(taskID, reason string) error
}

// Pool Worker 池
type Pool struct {
	size      int
	taskChan  chan *TaskWrapper
	semaphore chan struct{}
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc

	// 依赖
	repo              *repository.DownloadRepository
	executor          *ytdlp.Executor
	pathGenerator     *storage.PathGenerator
	fileManager       *storage.FileManager
	progressPublisher *ProgressPublisher
	assetClient       AssetClientInterface // 新增：Asset 客户端

	// 配置
	storageCfg    *config.StorageConfig
	retryCfg      *config.RetryConfig
	youtubePolicy platformpolicy.YouTubePolicy
}

// TaskWrapper 任务包装器
type TaskWrapper struct {
	Task     *models.DownloadTask
	Callback func(error)
}

// NewPool 创建 Worker 池
func NewPool(
	cfg *config.WorkerConfig,
	storageCfg *config.StorageConfig,
	retryCfg *config.RetryConfig,
	repo *repository.DownloadRepository,
	executor *ytdlp.Executor,
	pathGenerator *storage.PathGenerator,
	fileManager *storage.FileManager,
	progressPublisher *ProgressPublisher,
	assetClient AssetClientInterface, // 新增：Asset 客户端（可选）
	youtubePolicy platformpolicy.YouTubePolicy,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		size:              cfg.PoolSize,
		taskChan:          make(chan *TaskWrapper, cfg.PoolSize*2),
		semaphore:         make(chan struct{}, cfg.MaxConcurrent),
		ctx:               ctx,
		cancel:            cancel,
		repo:              repo,
		executor:          executor,
		pathGenerator:     pathGenerator,
		fileManager:       fileManager,
		progressPublisher: progressPublisher,
		assetClient:       assetClient, // 新增
		storageCfg:        storageCfg,
		retryCfg:          retryCfg,
		youtubePolicy:     youtubePolicy,
	}

	return pool
}

// Start 启动 Worker 池
func (p *Pool) Start() {
	log.Printf("[WorkerPool] Starting %d workers", p.size)
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop 停止 Worker 池
func (p *Pool) Stop() {
	log.Println("[WorkerPool] Stopping workers...")
	p.cancel()
	close(p.taskChan)
	p.wg.Wait()
	log.Println("[WorkerPool] All workers stopped")
}

// Submit 提交任务
func (p *Pool) Submit(task *models.DownloadTask, callback func(error)) {
	log.Printf("[WorkerPool] Submitting task %s to channel (current queue size: %d/%d)",
		task.TaskID, len(p.taskChan), cap(p.taskChan))
	wrapper := &TaskWrapper{
		Task:     task,
		Callback: callback,
	}

	select {
	case p.taskChan <- wrapper:
		log.Printf("[WorkerPool] ✓ Task %s submitted successfully", task.TaskID)
	case <-p.ctx.Done():
		log.Printf("[WorkerPool] ❌ Failed to submit task %s: context cancelled", task.TaskID)
		if callback != nil {
			callback(p.ctx.Err())
		}
	}
}

// worker 工作协程
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	log.Printf("[Worker %d] Started", id)

	for {
		select {
		case wrapper, ok := <-p.taskChan:
			if !ok {
				log.Printf("[Worker %d] Channel closed, exiting", id)
				return
			}

			// 获取信号量
			select {
			case p.semaphore <- struct{}{}:
			case <-p.ctx.Done():
				return
			}

			log.Printf("[Worker %d] ▶ Starting to process task: %s (URL: %s)", id, wrapper.Task.TaskID, wrapper.Task.URL)
			startTime := time.Now()
			err := p.processTask(wrapper.Task)
			duration := time.Since(startTime)

			if err != nil {
				log.Printf("[Worker %d] ❌ Task %s failed after %v: %v", id, wrapper.Task.TaskID, duration, err)
			} else {
				log.Printf("[Worker %d] ✓ Task %s completed successfully in %v", id, wrapper.Task.TaskID, duration)
			}

			// 释放信号量
			<-p.semaphore

			if wrapper.Callback != nil {
				wrapper.Callback(err)
			}

		case <-p.ctx.Done():
			log.Printf("[Worker %d] Context cancelled, exiting", id)
			return
		}
	}
}

// processTask 处理下载任务
func (p *Pool) processTask(task *models.DownloadTask) error {
	ctx := p.ctx
	taskID := task.TaskID

	log.Printf("[Worker] [Task %s] Step 1/10: Updating status to processing...", taskID)
	// 1. 更新状态为处理中
	if err := p.repo.UpdateProcessing(ctx, taskID); err != nil {
		log.Printf("[Worker] [Task %s] ❌ Failed to update status to processing: %v", taskID, err)
	} else {
		log.Printf("[Worker] [Task %s] ✓ Status updated to processing", taskID)
	}

	log.Printf("[Worker] [Task %s] Step 2/10: Checking disk space...", taskID)
	// 2. 检查磁盘空间
	sufficient, err := p.fileManager.IsDiskSpaceSufficient(p.storageCfg.BasePath, 90)
	if err != nil {
		log.Printf("[Worker] [Task %s] ❌ Failed to check disk space: %v", taskID, err)
		return p.handleError(ctx, task, err)
	}
	if !sufficient {
		log.Printf("[Worker] [Task %s] ❌ Insufficient disk space", taskID)
		return p.handleError(ctx, task, ErrInsufficientSpace)
	}
	log.Printf("[Worker] [Task %s] ✓ Disk space sufficient", taskID)

	log.Printf("[Worker] [Task %s] Step 3/10: Getting proxy IP...", taskID)
	// 3. 使用 Parser 阶段传递的 proxy，确保解析和下载代理一致
	proxyURL := task.ProxyURL
	if proxyURL == "" {
		log.Printf("[Worker] [Task %s] ✓ No proxy lease attached, using direct connection", taskID)
	} else {
		log.Printf("[Worker] [Task %s] ✓ Using proxy from parser: %s (lease_id=%s, expire_at=%s)", taskID, redact.ProxyURL(proxyURL), task.ProxyLeaseID, task.ProxyExpireAt)
	}

	log.Printf("[Worker] [Task %s] Step 4/10: Generating output path...", taskID)
	// 4. 生成文件路径
	outputPath, err := p.pathGenerator.GeneratePath(task)
	if err != nil {
		log.Printf("[Worker] [Task %s] ❌ Failed to generate path: %v", taskID, err)
		return p.handleError(ctx, task, err)
	}
	log.Printf("[Worker] [Task %s] ✓ Output path: %s", taskID, outputPath)

	log.Printf("[Worker] [Task %s] Step 5/10: Setting up progress callback...", taskID)
	// 5. 设置进度回调（带阶段跟踪）
	needsMerge := ytdlp.NeedsMerge(task)
	downloadRound := 0
	lastRawPercent := 0.0
	accumulatedIngressBytes := int64(0)
	currentRoundPeakBytes := int64(0)
	log.Printf("[Worker] [Task %s] NeedsMerge: %v", taskID, needsMerge)

	progressCallback := func(event *ytdlp.OutputEvent) {
		switch event.Type {
		case "merger":
			// 合流阶段
			phase := ytdlp.PhaseMerging
			percent := 85.0
			log.Printf("[Worker] [Task %s] Phase: %s, Percent: %.1f%%", taskID, phase, percent)
			if err := p.progressPublisher.PublishPhase(ctx, taskID, phase, percent); err != nil {
				log.Printf("[Worker] [Task %s] ⚠ Failed to publish merger phase: %v", taskID, err)
			}

		case "progress":
			raw := event.Progress.Percent

			// 检测是否进入新一轮下载（进度大幅回退说明开始下载第二个流）
			if raw < lastRawPercent-10 && lastRawPercent > 90 {
				accumulatedIngressBytes += currentRoundPeakBytes
				currentRoundPeakBytes = 0
				downloadRound++
				log.Printf("[Worker] [Task %s] Detected new download round: %d", taskID, downloadRound)
			}
			lastRawPercent = raw
			if event.Progress.DownloadedBytes > currentRoundPeakBytes {
				currentRoundPeakBytes = event.Progress.DownloadedBytes
			}

			// 确定当前阶段
			var phase ytdlp.DownloadPhase
			if !needsMerge {
				phase = ytdlp.PhaseDownloading
			} else if downloadRound == 0 {
				phase = ytdlp.PhaseDownloadingVideo
			} else {
				phase = ytdlp.PhaseDownloadingAudio
			}

			// 计算加权整体进度
			overall := calcOverallPercent(raw, phase, needsMerge)

			log.Printf("[Worker] [Task %s] Progress: %.1f%% (raw) → %.1f%% (overall), Phase: %s, Speed: %s, ETA: %s",
				taskID, raw, overall, phase, event.Progress.Speed, event.Progress.ETA)
			if err := p.progressPublisher.PublishDownloading(ctx, taskID, event.Progress, phase, overall); err != nil {
				log.Printf("[Worker] [Task %s] ⚠ Failed to publish progress: %v", taskID, err)
			}
		}
	}

	log.Printf("[Worker] [Task %s] Step 6/10: Starting download...", taskID)
	if err := p.progressPublisher.PublishStarted(ctx, taskID, "Download started"); err != nil {
		log.Printf("[Worker] [Task %s] ⚠ Failed to publish initial progress: %v", taskID, err)
	}

	// 6. 获取 Cookie 并执行下载
	var cookieFile string
	platform := task.Metadata.Platform
	log.Printf("[Worker] [Task %s] Download access policy: platform=%s youtube_cookie_disabled=%t has_proxy=%t", taskID, platform, platformpolicy.IsYouTubePlatform(platform) && p.youtubePolicy.CookiesDisabled(), proxyURL != "")

	// 如果有 CookieID，从 Asset Service 获取 cookie 内容
	if platformpolicy.IsYouTubePlatform(platform) && p.youtubePolicy.CookiesDisabled() {
		log.Printf("[Worker] [Task %s] youtube_cookie_disabled=true, skipping cookie fetch", taskID)
	} else if task.CookieID > 0 && p.assetClient != nil {
		log.Printf("[Worker] [Task %s] Getting cookie content for ID: %d", taskID, task.CookieID)
		var err error
		cookieFile, err = p.assetClient.GetCookieContent(task.CookieID, platform, taskID)
		if err != nil {
			log.Printf("[Worker] [Task %s] ⚠ Failed to get cookie: %v (continuing without cookie)", taskID, err)
		} else if cookieFile != "" {
			log.Printf("[Worker] [Task %s] ✓ Got cookie file: %s", taskID, cookieFile)
			// 确保下载完成后清理 cookie 文件
			defer func() {
				if cleanErr := p.assetClient.CleanupCookieFile(cookieFile); cleanErr != nil {
					log.Printf("[Worker] [Task %s] ⚠ Failed to cleanup cookie file: %v", taskID, cleanErr)
				}
			}()
		}
	}

	downloadErr := p.executor.Download(ctx, task, proxyURL, outputPath, cookieFile, progressCallback)

	if proxyURL != "" && task.ProxyLeaseID != "" && p.assetClient != nil {
		success := downloadErr == nil
		if reportErr := p.assetClient.ReportProxyUsage(task.TaskID, task.ProxyLeaseID, "download", success); reportErr != nil {
			log.Printf("[Worker] [Task %s] ⚠ Failed to report proxy usage: %v", taskID, reportErr)
		}
	}

	// 报告 Cookie 使用结果
	if platformpolicy.IsYouTubePlatform(platform) && p.youtubePolicy.CookiesDisabled() {
		log.Printf("[Worker] [Task %s] youtube_cookie_disabled=true, skipping cookie usage report", taskID)
	} else if task.CookieID > 0 && p.assetClient != nil {
		success := downloadErr == nil
		if reportErr := p.assetClient.ReportCookieUsage(task.CookieID, success, taskID); reportErr != nil {
			log.Printf("[Worker] [Task %s] ⚠ Failed to report cookie usage: %v", taskID, reportErr)
		}
	}

	if downloadErr != nil {
		log.Printf("[Worker] [Task %s] ❌ Download failed: %v", taskID, downloadErr)
		return p.handleError(ctx, task, downloadErr)
	}
	log.Printf("[Worker] [Task %s] ✓ Download completed", taskID)
	actualIngressBytes := accumulatedIngressBytes + currentRoundPeakBytes

	// 发布 processing 阶段
	processingStart := 85.0
	if needsMerge {
		processingStart = 92.0
	}
	if err := p.progressPublisher.PublishPhase(ctx, taskID, ytdlp.PhaseProcessing, processingStart); err != nil {
		log.Printf("[Worker] [Task %s] ⚠ Failed to publish processing phase: %v", taskID, err)
	}

	log.Printf("[Worker] [Task %s] Step 7/10: Calculating file information...", taskID)
	// 7. 计算文件信息
	fileSize, err := p.fileManager.GetFileSize(outputPath)
	if err != nil {
		log.Printf("[Worker] [Task %s] ❌ Failed to get file size: %v", taskID, err)
		return p.handleError(ctx, task, err)
	}
	log.Printf("[Worker] [Task %s] ✓ File size: %d bytes (%.2f MB)", taskID, fileSize, float64(fileSize)/1024/1024)

	log.Printf("[Worker] [Task %s] Step 8/10: Calculating MD5 hash...", taskID)
	fileHash, err := p.fileManager.CalculateMD5(outputPath)
	if err != nil {
		log.Printf("[Worker] [Task %s] ❌ Failed to calculate MD5: %v", taskID, err)
		return p.handleError(ctx, task, err)
	}
	log.Printf("[Worker] [Task %s] ✓ File hash: %s", taskID, fileHash)

	fileName := filepath.Base(outputPath)
	log.Printf("[Worker] [Task %s] ✓ File name: %s", taskID, fileName)

	if actualIngressBytes <= 0 {
		switch {
		case task.SelectedFormat != nil && task.SelectedFormat.Filesize > 0:
			actualIngressBytes = task.SelectedFormat.Filesize
		case fileSize > 0:
			actualIngressBytes = fileSize
		}
	}
	log.Printf("[Worker] [Task %s] ✓ Actual ingress bytes: %d", taskID, actualIngressBytes)

	log.Printf("[Worker] [Task %s] Step 9/10: Setting expiration time...", taskID)
	// 8. 计算过期时间 (仅 quick_download 模式)
	var expireAt *time.Time
	if task.Mode == "quick_download" {
		t := time.Now().Add(time.Duration(p.storageCfg.TmpTTL) * time.Second)
		expireAt = &t
		log.Printf("[Worker] [Task %s] ✓ Expiration set to: %v (TTL: %ds)", taskID, t, p.storageCfg.TmpTTL)
	} else {
		log.Printf("[Worker] [Task %s] ✓ No expiration (permanent storage)", taskID)
	}

	log.Printf("[Worker] [Task %s] Step 10/10: Updating database...", taskID)
	// 9. 更新数据库
	if err := p.repo.UpdateComplete(ctx, taskID, outputPath, fileName, fileHash, fileSize, expireAt); err != nil {
		log.Printf("[Worker] [Task %s] ❌ Failed to update database: %v", taskID, err)
		return p.handleError(ctx, task, err)
	}
	log.Printf("[Worker] [Task %s] ✓ Database updated", taskID)

	if p.assetClient != nil {
		if err := p.assetClient.CaptureIngressUsage(taskID, actualIngressBytes); err != nil {
			log.Printf("[Worker] [Task %s] ❌ Failed to capture ingress usage: %v", taskID, err)
			return p.handleError(ctx, task, err)
		}
		log.Printf("[Worker] [Task %s] ✓ Ingress usage captured", taskID)

		if err := p.assetClient.UpdateHistoryCompleted(taskID, outputPath, fileName, fileHash, fileSize, expireAt != nil); err != nil {
			log.Printf("[Worker] [Task %s] ❌ Failed to sync completed history to asset service: %v", taskID, err)
			return p.handleError(ctx, task, err)
		}
		log.Printf("[Worker] [Task %s] ✓ Asset history synced", taskID)
	}

	// 10. 发布完成消息
	log.Printf("[Worker] [Task %s] Publishing completion message...", taskID)
	if err := p.progressPublisher.PublishCompleted(ctx, taskID, "Download completed"); err != nil {
		log.Printf("[Worker] [Task %s] ⚠ Failed to publish completion: %v", taskID, err)
	} else {
		log.Printf("[Worker] [Task %s] ✓ Completion message published", taskID)
	}

	log.Printf("[Worker] [Task %s] ✅ All steps completed successfully - File: %s, Size: %d bytes", taskID, outputPath, fileSize)
	return nil
}

// handleError 处理错误
func (p *Pool) handleError(ctx context.Context, task *models.DownloadTask, err error) error {
	taskID := task.TaskID
	log.Printf("[Worker] [Task %s] ❌ Handling error: %v", taskID, err)
	log.Printf("[Worker] [Task %s] Task details - URL: %s, Mode: %s, Quality: %s, Format: %s",
		taskID, task.URL, task.Mode, task.Quality, task.Format)

	// 发布失败消息
	log.Printf("[Worker] [Task %s] Publishing failure message...", taskID)
	if pubErr := p.progressPublisher.PublishFailed(ctx, taskID, err.Error()); pubErr != nil {
		log.Printf("[Worker] [Task %s] ⚠ Failed to publish error: %v", taskID, pubErr)
	} else {
		log.Printf("[Worker] [Task %s] ✓ Failure message published", taskID)
	}

	// 更新数据库状态
	log.Printf("[Worker] [Task %s] Updating database with failed status...", taskID)
	if dbErr := p.repo.UpdateFailed(ctx, taskID, err.Error(), 0); dbErr != nil {
		log.Printf("[Worker] [Task %s] ⚠ Failed to update failed status in DB: %v", taskID, dbErr)
	} else {
		log.Printf("[Worker] [Task %s] ✓ Database updated with failed status", taskID)
	}

	if p.assetClient != nil {
		if releaseErr := p.assetClient.ReleaseInitialDownload(taskID, err.Error()); releaseErr != nil {
			log.Printf("[Worker] [Task %s] ⚠ Failed to release initial billing hold: %v", taskID, releaseErr)
		} else {
			log.Printf("[Worker] [Task %s] ✓ Initial billing hold released", taskID)
		}

		if syncErr := p.assetClient.UpdateHistoryFailed(taskID, err.Error()); syncErr != nil {
			log.Printf("[Worker] [Task %s] ⚠ Failed to sync failed status to asset service: %v", taskID, syncErr)
		} else {
			log.Printf("[Worker] [Task %s] ✓ Failed status synced to asset service", taskID)
		}
	}

	return err
}

// calcOverallPercent 计算加权整体进度
func calcOverallPercent(rawPercent float64, phase ytdlp.DownloadPhase, needsMerge bool) float64 {
	if !needsMerge {
		// 单流：[0–85%]
		return rawPercent * 0.85
	}
	switch phase {
	case ytdlp.PhaseDownloadingVideo:
		// 视频流：[0–65%]
		return rawPercent * 0.65
	case ytdlp.PhaseDownloadingAudio:
		// 音频流：[65–85%]
		return 65.0 + rawPercent*0.20
	default:
		return rawPercent * 0.85
	}
}
