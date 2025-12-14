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
	wrapper := &TaskWrapper{
		Task:     task,
		Callback: callback,
	}

	select {
	case p.taskChan <- wrapper:
		log.Printf("[WorkerPool] Task submitted: %s", task.TaskID)
	case <-p.ctx.Done():
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

			log.Printf("[Worker %d] Processing task: %s", id, wrapper.Task.TaskID)
			err := p.processTask(wrapper.Task)

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

	// 1. 更新状态为处理中
	if err := p.repo.UpdateProcessing(ctx, taskID); err != nil {
		log.Printf("[Worker] Failed to update status: %v", err)
	}

	// 2. 检查磁盘空间
	sufficient, err := p.fileManager.IsDiskSpaceSufficient(p.storageCfg.BasePath, 90)
	if err != nil {
		return p.handleError(ctx, task, err)
	}
	if !sufficient {
		return p.handleError(ctx, task, ErrInsufficientSpace)
	}

	// 3. 获取代理 IP
	proxyURL, err := p.proxyProvider.GetProxyWithRetry(ctx)
	if err != nil {
		return p.handleError(ctx, task, err)
	}

	// 4. 生成文件路径
	outputPath, err := p.pathGenerator.GeneratePath(task)
	if err != nil {
		return p.handleError(ctx, task, err)
	}

	// 5. 设置进度回调
	p.executor.SetProgressCallback(func(progress *models.Progress) {
		if err := p.progressPublisher.PublishDownloading(ctx, taskID, progress); err != nil {
			log.Printf("[Worker] Failed to publish progress: %v", err)
		}
	})

	// 6. 执行下载
	if err := p.executor.Download(ctx, task, proxyURL, outputPath); err != nil {
		return p.handleError(ctx, task, err)
	}

	// 7. 计算文件信息
	fileSize, err := p.fileManager.GetFileSize(outputPath)
	if err != nil {
		return p.handleError(ctx, task, err)
	}

	fileHash, err := p.fileManager.CalculateMD5(outputPath)
	if err != nil {
		return p.handleError(ctx, task, err)
	}

	fileName := filepath.Base(outputPath)

	// 8. 计算过期时间 (仅 quick_download 模式)
	var expireAt *time.Time
	if task.Mode == "quick_download" {
		t := time.Now().Add(time.Duration(p.storageCfg.TmpTTL) * time.Second)
		expireAt = &t
	}

	// 9. 更新数据库
	if err := p.repo.UpdateComplete(ctx, taskID, outputPath, fileName, fileHash, fileSize, expireAt); err != nil {
		return p.handleError(ctx, task, err)
	}

	// 10. 发布完成消息
	if err := p.progressPublisher.PublishCompleted(ctx, taskID, "Download completed"); err != nil {
		log.Printf("[Worker] Failed to publish completion: %v", err)
	}

	log.Printf("[Worker] Task completed: %s, file: %s, size: %d bytes", taskID, outputPath, fileSize)
	return nil
}

// handleError 处理错误
func (p *Pool) handleError(ctx context.Context, task *models.DownloadTask, err error) error {
	taskID := task.TaskID
	log.Printf("[Worker] Task %s failed: %v", taskID, err)

	// 发布失败消息
	if pubErr := p.progressPublisher.PublishFailed(ctx, taskID, err.Error()); pubErr != nil {
		log.Printf("[Worker] Failed to publish error: %v", pubErr)
	}

	// 更新数据库状态
	if dbErr := p.repo.UpdateFailed(ctx, taskID, err.Error(), 0); dbErr != nil {
		log.Printf("[Worker] Failed to update failed status: %v", dbErr)
	}

	return err
}
