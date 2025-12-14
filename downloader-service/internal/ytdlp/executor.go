package ytdlp

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
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
	progressCallback    func(*models.Progress)
}

// NewExecutor 创建 yt-dlp 执行器
func NewExecutor(cfg *config.YtDLPConfig) *Executor {
	return &Executor{
		binaryPath:          cfg.BinaryPath,
		timeout:             time.Duration(cfg.Timeout) * time.Second,
		concurrentFragments: cfg.ConcurrentFragments,
	}
}

// SetProgressCallback 设置进度回调
func (e *Executor) SetProgressCallback(callback func(*models.Progress)) {
	e.progressCallback = callback
}

// Download 执行下载
func (e *Executor) Download(ctx context.Context, task *models.DownloadTask, proxyURL, outputPath string) error {
	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// 构建命令
	cmd := e.buildCommand(task, proxyURL, outputPath)
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

	// 获取标准输出和标准错误
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 启动命令
	log.Printf("[YtDLP] Starting download: %s", task.URL)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// 读取错误输出
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stderrOutput.WriteString(line + "\n")
			log.Printf("[YtDLP] stderr: %s", line)
		}
	}()

	// 解析进度输出
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		line := scanner.Text()
		progress := parseProgress(line)
		if progress != nil && e.progressCallback != nil {
			e.progressCallback(progress)
		}
	}

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("download timeout after %v", e.timeout)
		}
		return fmt.Errorf("yt-dlp failed: %w, stderr: %s", err, stderrOutput.String())
	}

	log.Printf("[YtDLP] Download completed: %s", task.URL)
	return nil
}

// buildCommand 构建下载命令
func (e *Executor) buildCommand(task *models.DownloadTask, proxyURL, outputPath string) *exec.Cmd {
	args := []string{
		"--output", outputPath,
		"--merge-output-format", task.Format,
		"--progress", // 输出进度
		"--newline",  // 每次进度新行
		"--no-playlist",
		"--concurrent-fragments", fmt.Sprintf("%d", e.concurrentFragments),
	}

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
	log.Printf("[YtDLP] Command: %s %s", e.binaryPath, strings.Join(args, " "))
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
