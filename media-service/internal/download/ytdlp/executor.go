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

	"vasset/media-service/internal/download/config"
	"vasset/media-service/internal/download/models"
	"vasset/media-service/internal/platformpolicy"
	"vasset/media-service/internal/redact"
)

const (
	formatTracePrefix = "[format-trace]"
	fileTracePrefix   = "[file-trace]"
)

// DownloadPhase 下载阶段
type DownloadPhase string

const (
	PhaseDownloadingVideo DownloadPhase = "downloading_video"
	PhaseDownloadingAudio DownloadPhase = "downloading_audio"
	PhaseDownloading      DownloadPhase = "downloading"
	PhaseMerging          DownloadPhase = "merging"
	PhaseProcessing       DownloadPhase = "processing"
)

// OutputEvent 表示从 yt-dlp stdout 解析出的一个事件
type OutputEvent struct {
	Type     string           // "progress" 或 "merger"
	Progress *models.Progress // Type=="progress" 时有值
}

// NeedsMerge 判断任务是否需要音视频合流
func NeedsMerge(task *models.DownloadTask) bool {
	sel := task.SelectedFormat
	if sel == nil {
		// 无精确选择，走 buildFormatString 路径
		switch task.Quality {
		case "best", "audio", "":
			return false
		default:
			return true // 720p/1080p 等大概率需要 video+audio
		}
	}
	// 有精确选择：有视频但无音频 → 需要合流
	return hasVideo(sel) && !hasAudio(sel)
}

// Executor yt-dlp 执行器
type Executor struct {
	binaryPath          string
	timeout             time.Duration
	concurrentFragments int
	cookiesDir          string
	defaultArgs         []string
	platformArgs        map[string][]string
	youtubePolicy       platformpolicy.YouTubePolicy
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
		youtubePolicy:       cfg.YouTube,
	}
}

// Download 执行下载
// cookieFile: 可选的 cookie 文件路径，如果非空则优先使用
func (e *Executor) Download(ctx context.Context, task *models.DownloadTask, proxyURL, outputPath, cookieFile string, callback func(*OutputEvent)) error {
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
		if strings.HasPrefix(line, formatTracePrefix) || strings.HasPrefix(line, fileTracePrefix) {
			log.Printf("[YtDLP] [Task %s] %s", task.TaskID, line)
			continue
		}
		event := parseOutput(line)
		if event != nil && callback != nil {
			callback(event)
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
	if platformpolicy.IsYouTubePlatform(platform) {
		args = append(args, e.youtubePolicy.Args...)
		log.Printf("[YtDLP] [Task %s] Applied shared YouTube args: %d", task.TaskID, len(e.youtubePolicy.Args))
	} else if platformArgs, ok := e.platformArgs[platform]; ok && len(platformArgs) > 0 {
		args = append(args, platformArgs...)
		log.Printf("[YtDLP] [Task %s] Added %d platform-specific args for %s", task.TaskID, len(platformArgs), platform)
	}

	// 添加 cookies：优先使用传入的 cookieFile，否则回退到自动检测
	if platformpolicy.IsYouTubePlatform(platform) && e.youtubePolicy.CookiesDisabled() {
		log.Printf("[YtDLP] [Task %s] youtube_cookie_disabled=true, skipping cookie injection", task.TaskID)
	} else if cookieFile != "" {
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
	args = append(args,
		"--print", e.buildFormatTraceTemplate(),
		"--print", e.buildFileTraceTemplate(),
	)

	// 添加输出格式
	format := e.resolveOutputFormat(task)
	if e.shouldSetMergeOutputFormat(task) {
		args = append(args, "--merge-output-format", format)
	}

	// 添加代理
	if proxyURL != "" {
		args = append(args, "--proxy", proxyURL)
	}

	// 添加格式选择
	if formatSelector := e.buildRequestedFormat(task); formatSelector != "" {
		args = append(args, "--format", formatSelector)
	} else if task.Quality != "" {
		args = append(args, "--format", e.buildFormatString(task.Quality, format))
	}

	// 添加 URL
	args = append(args, task.URL)

	cmd := exec.Command(e.binaryPath, args...)
	log.Printf("[YtDLP] [Task %s] Command: %s %s", task.TaskID, e.binaryPath, strings.Join(redact.ProxyArgs(args), " "))
	return cmd
}

// buildFormatString 构建格式选择字符串
func (e *Executor) buildFormatString(quality, format string) string {
	// 根据质量选择格式
	// 例如: bestvideo[height<=1080]+bestaudio[ext=m4a]/best[height<=1080]
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

	preferredVideo := fmt.Sprintf("bestvideo[height<=%s]", height)
	if format != "" {
		preferredVideo = fmt.Sprintf("bestvideo[height<=%s][ext=%s]", height, format)
	}
	fallbackVideo := fmt.Sprintf("bestvideo[height<=%s]", height)
	preferredAudio := e.buildCompatibleAudioSelector(format)
	bestSelector := fmt.Sprintf("best[height<=%s]", height)
	if format != "" {
		bestSelector = fmt.Sprintf("best[height<=%s][ext=%s]/best[height<=%s]", height, format, height)
	}

	if preferredAudio == "bestaudio" {
		return fmt.Sprintf("%s+bestaudio/%s+bestaudio/%s", preferredVideo, fallbackVideo, bestSelector)
	}

	return fmt.Sprintf("%s+%s/%s+bestaudio/%s+bestaudio/%s", preferredVideo, preferredAudio, preferredVideo, fallbackVideo, bestSelector)
}

func (e *Executor) resolveOutputFormat(task *models.DownloadTask) string {
	if task.Format != "" {
		return task.Format
	}
	if task.SelectedFormat != nil && task.SelectedFormat.Extension != "" {
		return task.SelectedFormat.Extension
	}
	return "mp4"
}

func (e *Executor) shouldSetMergeOutputFormat(task *models.DownloadTask) bool {
	if task.SelectedFormat == nil {
		return true
	}
	if !hasVideo(task.SelectedFormat) {
		return false
	}
	return !hasAudio(task.SelectedFormat)
}

func (e *Executor) buildRequestedFormat(task *models.DownloadTask) string {
	selected := task.SelectedFormat
	if selected == nil {
		if task.FormatID == "" {
			return ""
		}
		return fmt.Sprintf("%s/best", task.FormatID)
	}
	if selected.FormatID == "" {
		return ""
	}

	if !hasVideo(selected) {
		return selected.FormatID
	}
	if hasAudio(selected) {
		return selected.FormatID
	}

	audioSelector := e.buildCompatibleAudioSelector(e.resolveOutputFormat(task))
	if audioSelector == "bestaudio" {
		return fmt.Sprintf("%s+bestaudio/%s/best", selected.FormatID, selected.FormatID)
	}
	return fmt.Sprintf("%s+%s/%s+bestaudio/%s/best", selected.FormatID, audioSelector, selected.FormatID, selected.FormatID)
}

func (e *Executor) buildCompatibleAudioSelector(format string) string {
	switch strings.ToLower(format) {
	case "mp4", "m4a", "mov":
		return "bestaudio[ext=m4a]"
	case "webm":
		return "bestaudio[ext=webm]"
	default:
		return "bestaudio"
	}
}

func (e *Executor) buildFormatTraceTemplate() string {
	return "before_dl:" + formatTracePrefix + " " +
		"format_id=%(format_id)s " +
		"ext=%(ext)s " +
		"resolution=%(resolution)s " +
		"fps=%(fps)s " +
		"vcodec=%(vcodec)s " +
		"acodec=%(acodec)s " +
		"req0=%(requested_formats.0.format_id)s " +
		"req1=%(requested_formats.1.format_id)s"
}

func (e *Executor) buildFileTraceTemplate() string {
	return "after_move:" + fileTracePrefix + " filepath=%(filepath)s"
}

func hasVideo(selected *models.SelectedFormat) bool {
	return selected != nil && selected.VideoCodec != "" && selected.VideoCodec != "none"
}

func hasAudio(selected *models.SelectedFormat) bool {
	return selected != nil && selected.AudioCodec != "" && selected.AudioCodec != "none"
}

// parseOutput 解析 yt-dlp stdout 输出为事件
func parseOutput(line string) *OutputEvent {
	// 识别 [Merger] 行
	if strings.Contains(line, "[Merger]") {
		return &OutputEvent{Type: "merger"}
	}
	// 识别 [download] 进度行
	progress := parseDownloadProgress(line)
	if progress != nil {
		return &OutputEvent{Type: "progress", Progress: progress}
	}
	return nil
}

// parseDownloadProgress 解析 yt-dlp 进度输出
// 格式: [download]  45.2% of 100.00MiB at 2.50MiB/s ETA 00:22
func parseDownloadProgress(line string) *models.Progress {
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
