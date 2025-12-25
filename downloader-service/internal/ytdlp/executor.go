package ytdlp

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vasset/downloader-service/internal/config"
	"vasset/downloader-service/internal/models"
)

// Executor yt-dlp 执行器
type Executor struct {
	binaryPath          string
	timeout             time.Duration
	concurrentFragments int
	cookiesDir          string
	defaultArgs         []string
	platformArgs        map[string][]string
	progressCallback    func(*models.Progress)
}

// NewExecutor 创建 yt-dlp 执行器
func NewExecutor(cfg *config.YtDLPConfig) *Executor {
	return &Executor{
		binaryPath:          cfg.BinaryPath,
		timeout:             time.Duration(cfg.Timeout) * time.Second,
		concurrentFragments: cfg.ConcurrentFragments,
		cookiesDir:          cfg.CookiesDir,
		defaultArgs:         cfg.DefaultArgs,
		platformArgs:        cfg.PlatformArgs,
	}
}

// SetProgressCallback 设置进度回调
func (e *Executor) SetProgressCallback(callback func(*models.Progress)) {
	e.progressCallback = callback
}

// Download 执行下载
// cookieFile: 可选的 cookie 文件路径，如果非空则优先使用
func (e *Executor) Download(ctx context.Context, task *models.DownloadTask, proxyURL, outputPath, cookieFile string) error {
	log.Printf("[YtDLP] [Task %s] Preparing download for URL: %s", task.TaskID, task.URL)
	log.Printf("[YtDLP] [Task %s] Download parameters - Quality: %s, Format: %s, Output: %s, Cookie: %s",
		task.TaskID, task.Quality, task.Format, outputPath, cookieFile)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	log.Printf("[YtDLP] [Task %s] Download timeout set to: %v", task.TaskID, e.timeout)

	// 构建命令
	log.Printf("[YtDLP] [Task %s] Building yt-dlp command...", task.TaskID)
	cmd := e.buildCommand(task, proxyURL, outputPath, cookieFile)
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// 获取标准输出和标准错误
	log.Printf("[YtDLP] [Task %s] Setting up stdout/stderr pipes...", task.TaskID)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[YtDLP] [Task %s] ❌ Failed to get stdout pipe: %v", task.TaskID, err)
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("[YtDLP] [Task %s] ❌ Failed to get stderr pipe: %v", task.TaskID, err)
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 启动命令
	log.Printf("[YtDLP] [Task %s] ▶ Starting yt-dlp download: %s", task.TaskID, task.URL)
	if err := cmd.Start(); err != nil {
		log.Printf("[YtDLP] [Task %s] ❌ Failed to start yt-dlp: %v", task.TaskID, err)
		return fmt.Errorf("failed to start yt-dlp: %w", err)
	}
	log.Printf("[YtDLP] [Task %s] ✓ yt-dlp process started", task.TaskID)

	// 读取错误输出
	log.Printf("[YtDLP] [Task %s] Starting stderr reader...", task.TaskID)
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stderrOutput.WriteString(line + "\n")
			log.Printf("[YtDLP] [Task %s] stderr: %s", task.TaskID, line)
		}
	}()

	// 解析进度输出
	log.Printf("[YtDLP] [Task %s] Starting stdout reader for progress...", task.TaskID)
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		progress := parseProgress(line)
		if progress != nil && e.progressCallback != nil {
			e.progressCallback(progress)
		}
	}

	// 等待命令完成
	log.Printf("[YtDLP] [Task %s] Waiting for yt-dlp to complete...", task.TaskID)
	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[YtDLP] [Task %s] ❌ Download timeout after %v", task.TaskID, e.timeout)
			return fmt.Errorf("download timeout after %v", e.timeout)
		}
		log.Printf("[YtDLP] [Task %s] ❌ yt-dlp failed: %v, stderr: %s", task.TaskID, err, stderrOutput.String())
		return fmt.Errorf("yt-dlp failed: %w, stderr: %s", err, stderrOutput.String())
	}

	log.Printf("[YtDLP] [Task %s] ✓ Download completed successfully: %s", task.TaskID, task.URL)
	return nil
}

// buildCommand 构建下载命令
// cookieFile: 优先使用此参数传入的 cookie 文件，为空则回退到自动检测
func (e *Executor) buildCommand(task *models.DownloadTask, proxyURL, outputPath, cookieFile string) *exec.Cmd {
	// 基础参数
	args := []string{
		"--output", outputPath,
		"--progress", // 输出进度
		"--newline",  // 每次进度新行
	}

	// 添加默认参数（从配置文件）
	if len(e.defaultArgs) > 0 {
		args = append(args, e.defaultArgs...)
		log.Printf("[YtDLP] [Task %s] Added %d default args", task.TaskID, len(e.defaultArgs))
	}

	// 检测平台并添加平台特定参数
	platform := detectPlatform(task.URL)
	log.Printf("[YtDLP] [Task %s] Detected platform: %s", task.TaskID, platform)
	if platformArgs, ok := e.platformArgs[platform]; ok && len(platformArgs) > 0 {
		args = append(args, platformArgs...)
		log.Printf("[YtDLP] [Task %s] Added %d platform-specific args for %s", task.TaskID, len(platformArgs), platform)
	}

	// 添加 cookies：优先使用传入的 cookieFile，否则回退到自动检测
	if cookieFile != "" {
		args = append(args, "--cookies", cookieFile)
		log.Printf("[YtDLP] [Task %s] Using provided cookie file: %s", task.TaskID, cookieFile)
	} else {
		// 回退到自动检测
		detectedCookie := e.getCookieFile(platform)
		if detectedCookie != "" {
			args = append(args, "--cookies", detectedCookie)
			log.Printf("[YtDLP] [Task %s] Using auto-detected cookie file: %s", task.TaskID, detectedCookie)
		}
	}

	// 添加并发分片下载
	args = append(args, "--concurrent-fragments", fmt.Sprintf("%d", e.concurrentFragments))

	// 添加输出格式
	format := task.Format
	if format == "" {
		format = "mp4" // 默认使用 mp4 格式
	}
	args = append(args, "--merge-output-format", format)

	// 添加代理
	if proxyURL != "" {
		args = append(args, "--proxy", proxyURL)
	}

	// 添加质量选择
	if task.Quality != "" {
		args = append(args, "--format", e.buildFormatString(task.Quality, task.Format))
	}

	// 添加 URL
	args = append(args, task.URL)

	cmd := exec.Command(e.binaryPath, args...)
	log.Printf("[YtDLP] [Task %s] Command: %s %s", task.TaskID, e.binaryPath, strings.Join(args, " "))
	return cmd
}

// buildFormatString 构建格式选择字符串
func (e *Executor) buildFormatString(quality, format string) string {
	// 根据质量选择格式
	// 例如: bestvideo[height<=1080]+bestaudio/best[height<=1080]
	height := ""
	switch quality {
	case "2160p", "4K":
		height = "2160"
	case "1440p":
		height = "1440"
	case "1080p":
		height = "1080"
	case "720p":
		height = "720"
	case "480p":
		height = "480"
	case "360p":
		height = "360"
	default:
		return "best"
	}

	return fmt.Sprintf("bestvideo[height<=%s]+bestaudio/best[height<=%s]", height, height)
}

// parseProgress 解析 yt-dlp 进度输出
// 格式: [download]  45.2% of 100.00MiB at 2.50MiB/s ETA 00:22
func parseProgress(line string) *models.Progress {
	if !strings.Contains(line, "[download]") {
		return nil
	}

	// 解析百分比
	percentRe := regexp.MustCompile(`(\d+\.?\d*)%`)
	percentMatch := percentRe.FindStringSubmatch(line)
	if len(percentMatch) < 2 {
		return nil
	}

	percent, err := strconv.ParseFloat(percentMatch[1], 64)
	if err != nil {
		return nil
	}

	progress := &models.Progress{
		Percent: percent,
	}

	// 解析速度
	speedRe := regexp.MustCompile(`at\s+(\d+\.?\d*\w+/s)`)
	speedMatch := speedRe.FindStringSubmatch(line)
	if len(speedMatch) >= 2 {
		progress.Speed = speedMatch[1]
	}

	// 解析 ETA
	etaRe := regexp.MustCompile(`ETA\s+(\d+:\d+)`)
	etaMatch := etaRe.FindStringSubmatch(line)
	if len(etaMatch) >= 2 {
		progress.ETA = etaMatch[1]
	}

	return progress
}

// detectPlatform 从 URL 检测平台
func detectPlatform(url string) string {
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return "youtube"
	}
	if strings.Contains(url, "bilibili.com") {
		return "bilibili"
	}
	if strings.Contains(url, "tiktok.com") {
		return "tiktok"
	}
	return "generic"
}

// getCookieFile 获取平台 cookie 文件
func (e *Executor) getCookieFile(platform string) string {
	if e.cookiesDir == "" {
		return ""
	}
	cookiePath := filepath.Join(e.cookiesDir, platform+".txt")
	if _, err := os.Stat(cookiePath); err == nil {
		log.Printf("[YtDLP] Found cookie file: %s", cookiePath)
		return cookiePath
	}
	return ""
}

// GetVideoInfo 获取视频信息(仅元数据,不下载)
func (e *Executor) GetVideoInfo(ctx context.Context, url string) (*models.Metadata, error) {
	args := []string{
		"--dump-json",
		"--no-download",
		"--no-playlist",
		url,
	}

	cmd := exec.CommandContext(ctx, e.binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	// 解析 JSON 输出
	// 这里简化处理,实际应该解析完整的 yt-dlp JSON 输出
	_ = output
	return &models.Metadata{}, nil
}
