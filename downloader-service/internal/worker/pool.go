package worker

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"vasset/downloader-service/internal/config"
	"vasset/downloader-service/internal/models"
	"vasset/downloader-service/internal/repository"
	"vasset/downloader-service/internal/storage"
	"vasset/downloader-service/internal/ytdlp"
)

// AssetClientInterface Asset 服务客户端接口
type AssetClientInterface interface {
	GetCookieContent(cookieID int64, platform, taskID string) (string, error)
	ReportCookieUsage(cookieID int64, success bool, taskID string) error
	CleanupCookieFile(cookieFile string) error
}

// ProxyProviderInterface 代理提供者接口
type ProxyProviderInterface interface {
	GetProxyWithRetry(ctx context.Context) (string, error)
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
	proxyProvider     ProxyProviderInterface
	executor          *ytdlp.Executor
	pathGenerator     *storage.PathGenerator
	fileManager       *storage.FileManager
	progressPublisher *ProgressPublisher
	assetClient       AssetClientInterface // 新增：Asset 客户端

	// 配置
	storageCfg *config.StorageConfig
	retryCfg   *config.RetryConfig
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
	proxyProvider ProxyProviderInterface,
	executor *ytdlp.Executor,
	pathGenerator *storage.PathGenerator,
	fileManager *storage.FileManager,
	progressPublisher *ProgressPublisher,
	assetClient AssetClientInterface, // 新增：Asset 客户端（可选）
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &Pool{
		size:              cfg.PoolSize,
		taskChan:          make(chan *TaskWrapper, cfg.PoolSize*2),
		semaphore:         make(chan struct{}, cfg.MaxConcurrent),
		ctx:               ctx,
		cancel:            cancel,
		repo:              repo,
		proxyProvider:     proxyProvider,
		executor:          executor,
		pathGenerator:     pathGenerator,
		fileManager:       fileManager,
		progressPublisher: progressPublisher,
		assetClient:       assetClient, // 新增
		storageCfg:        storageCfg,
		retryCfg:          retryCfg,
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
	// 3. 获取代理 IP - 优先使用 Parser 阶段传递的 proxy，确保一致性
	var proxyURL string
	if task.ProxyURL != "" {
		proxyURL = task.ProxyURL
		log.Printf("[Worker] [Task %s] ✓ Using proxy from parser: %s", taskID, proxyURL)
	} else {
		// 如果 Parser 没有传递 proxy，则独立获取
		var err error
		proxyURL, err = p.proxyProvider.GetProxyWithRetry(ctx)
		if err != nil {
			log.Printf("[Worker] [Task %s] ❌ Failed to get proxy: %v", taskID, err)
			return p.handleError(ctx, task, err)
		}
		if proxyURL != "" {
			log.Printf("[Worker] [Task %s] ✓ Got new proxy: %s", taskID, proxyURL)
		} else {
			log.Printf("[Worker] [Task %s] ✓ Using direct connection (no proxy)", taskID)
		}
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
	// 5. 设置进度回调
	p.executor.SetProgressCallback(func(progress *models.Progress) {
		log.Printf("[Worker] [Task %s] Progress: %.1f%%, Speed: %s, ETA: %s",
			taskID, progress.Percent, progress.Speed, progress.ETA)
		if err := p.progressPublisher.PublishDownloading(ctx, taskID, progress); err != nil {
			log.Printf("[Worker] [Task %s] ⚠ Failed to publish progress: %v", taskID, err)
		}
	})

	log.Printf("[Worker] [Task %s] Step 6/10: Starting download...", taskID)
	// 6. 获取 Cookie 并执行下载
	var cookieFile string
	platform := task.Metadata.Platform

	// 如果有 CookieID，从 Asset Service 获取 cookie 内容
	if task.CookieID > 0 && p.assetClient != nil {
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

	downloadErr := p.executor.Download(ctx, task, proxyURL, outputPath, cookieFile)

	// 报告 Cookie 使用结果
	if task.CookieID > 0 && p.assetClient != nil {
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

	return err
}
