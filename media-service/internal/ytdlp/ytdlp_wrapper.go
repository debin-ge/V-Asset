package ytdlp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"vasset/media-service/internal/config"
	"vasset/media-service/internal/redact"
	"vasset/media-service/internal/utils"
)

// VideoInfo yt-dlp返回的视频信息
type VideoInfo struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Duration    int64               `json:"duration"`
	Thumbnail   string              `json:"thumbnail"`
	Uploader    string              `json:"uploader"`
	UploadDate  string              `json:"upload_date"`
	ViewCount   int64               `json:"view_count"`
	Formats     []utils.VideoFormat `json:"formats"`
}

// Wrapper yt-dlp命令封装器
type Wrapper struct {
	binaryPath  string
	timeout     time.Duration
	proxy       string
	cookiesDir  string
	defaultArgs []string
}

// NewWrapper 创建yt-dlp封装器
func NewWrapper(cfg *config.YTDLPConfig) *Wrapper {
	return &Wrapper{
		binaryPath:  cfg.BinaryPath,
		timeout:     cfg.GetTimeout(),
		proxy:       cfg.Proxy,
		cookiesDir:  cfg.CookiesDir,
		defaultArgs: cfg.DefaultArgs,
	}
}

// buildArgs 构建命令参数
func (w *Wrapper) buildArgs(url string, cookieFile string, extraArgs []string) []string {
	args := []string{
		"--dump-json",
		"--skip-download",
	}

	// 添加默认参数
	args = append(args, w.defaultArgs...)

	// 添加代理 (如果配置了)
	if w.proxy != "" {
		args = append(args, "--proxy", w.proxy)
	}

	// 添加 cookie 文件 (如果存在)
	if cookieFile != "" {
		if _, err := os.Stat(cookieFile); err == nil {
			args = append(args, "--cookies", cookieFile)
		}
	}

	// 添加额外参数 (平台特定)
	args = append(args, extraArgs...)

	// 添加 URL
	args = append(args, url)

	return args
}

func formatCommandError(err error, output []byte) error {
	outputText := strings.TrimSpace(string(output))
	if outputText == "" {
		if err == nil {
			return utils.ErrYTDLPFailed
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			outputText = strings.TrimSpace(string(exitErr.Stderr))
		}
		if outputText == "" {
			outputText = err.Error()
		}
	}
	return fmt.Errorf("%w: %s", utils.MapYTDLPError(outputText), outputText)
}

func createEphemeralCacheDir(logPrefix string) (string, func()) {
	cacheDir, err := os.MkdirTemp("", "yt-dlp-cache-*")
	if err != nil {
		log.Printf("[%s] WARN: Failed to create isolated cache dir: %v", logPrefix, err)
		return "", func() {}
	}

	return cacheDir, func() {
		if removeErr := os.RemoveAll(cacheDir); removeErr != nil {
			log.Printf("[%s] WARN: Failed to cleanup cache dir %s: %v", logPrefix, cacheDir, removeErr)
		}
	}
}

func (w *Wrapper) commandContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, w.timeout)
}

func (w *Wrapper) executeJSONCommand(ctx context.Context, logPrefix string, args []string) ([]byte, error) {
	cacheDir, cleanupCacheDir := createEphemeralCacheDir(logPrefix)
	defer cleanupCacheDir()

	commandArgs := append([]string(nil), args...)
	if cacheDir != "" {
		commandArgs = append([]string{"--cache-dir", cacheDir}, commandArgs...)
		log.Printf("[%s] Using isolated cache dir: %s", logPrefix, cacheDir)
	}

	ctx, cancel := w.commandContext(ctx)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.binaryPath, commandArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("[%s] ERROR: Timeout after %v", logPrefix, w.timeout)
			return nil, utils.ErrTimeout
		}
		if ctx.Err() == context.Canceled {
			log.Printf("[%s] ERROR: Command canceled by parent context", logPrefix)
			return nil, utils.ErrTimeout
		}
		log.Printf("[%s] ERROR: Command failed: %v", logPrefix, err)
		log.Printf("[%s] Output: %s", logPrefix, string(output))
		return nil, formatCommandError(err, output)
	}

	return output, nil
}

// ExtractInfo 提取视频信息
func (w *Wrapper) ExtractInfo(ctx context.Context, url string, cookieFile string, extraArgs ...string) (*VideoInfo, error) {
	args := w.buildArgs(url, cookieFile, extraArgs)

	// 记录完整命令
	log.Printf("[YT-DLP] Executing command: %s %s", w.binaryPath, strings.Join(redact.ProxyArgs(args), " "))
	log.Printf("[YT-DLP] URL: %s", url)
	if cookieFile != "" {
		log.Printf("[YT-DLP] Cookie file: %s", cookieFile)
	}

	// 执行命令
	output, err := w.executeJSONCommand(ctx, "YT-DLP", args)
	if err != nil {
		return nil, err
	}

	log.Printf("[YT-DLP] Success, output length: %d bytes, %s", len(output), string(output))

	// 解析JSON输出
	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		log.Printf("[YT-DLP] ERROR: Failed to parse JSON: %v", err)
		log.Printf("[YT-DLP] Raw output: %s", string(output))
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	log.Printf("[YT-DLP] Parsed video: ID=%s, Title=%s, Formats=%d", info.ID, info.Title, len(info.Formats))
	return &info, nil
}

// ExtractInfoWithProxy 使用指定代理提取视频信息 (临时覆盖默认代理)
func (w *Wrapper) ExtractInfoWithProxy(ctx context.Context, url, proxyURL, cookieFile string, extraArgs ...string) (*VideoInfo, error) {
	args := []string{
		"--dump-json",
		"--skip-download",
	}

	// 添加默认参数 (排除代理相关)
	for i := 0; i < len(w.defaultArgs); i++ {
		if w.defaultArgs[i] == "--proxy" && i+1 < len(w.defaultArgs) {
			i++ // 跳过代理值
			continue
		}
		args = append(args, w.defaultArgs[i])
	}

	// 使用指定代理
	if proxyURL != "" {
		args = append(args, "--proxy", proxyURL)
	}

	// 添加 cookie 文件
	if cookieFile != "" {
		if _, err := os.Stat(cookieFile); err == nil {
			args = append(args, "--cookies", cookieFile)
		}
	}

	// 添加额外参数
	args = append(args, extraArgs...)
	args = append(args, url)

	// 记录完整命令
	log.Printf("[YT-DLP-PROXY] Executing command: %s %s", w.binaryPath, strings.Join(redact.ProxyArgs(args), " "))
	log.Printf("[YT-DLP-PROXY] URL: %s, Proxy: %s", url, redact.ProxyURL(proxyURL))
	if cookieFile != "" {
		log.Printf("[YT-DLP-PROXY] Cookie file: %s", cookieFile)
	}

	output, err := w.executeJSONCommand(ctx, "YT-DLP-PROXY", args)
	if err != nil {
		return nil, err
	}

	log.Printf("[YT-DLP-PROXY] Success, output length: %d bytes, %s", len(output), string(output))

	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		log.Printf("[YT-DLP-PROXY] ERROR: Failed to parse JSON: %v", err)
		log.Printf("[YT-DLP-PROXY] Raw output: %s", string(output))
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	log.Printf("[YT-DLP-PROXY] Parsed video: ID=%s, Title=%s, Formats=%d", info.ID, info.Title, len(info.Formats))
	return &info, nil
}

// Validate 验证URL是否可以解析
func (w *Wrapper) Validate(url string) error {
	args := []string{"--simulate", "--no-download"}

	// 添加代理
	if w.proxy != "" {
		args = append(args, "--proxy", w.proxy)
	}

	args = append(args, url)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.binaryPath, args...)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return utils.ErrTimeout
		}
		return utils.ErrVideoNotFound
	}

	return nil
}

// GetDefaultProxy 获取默认代理
func (w *Wrapper) GetDefaultProxy() string {
	return w.proxy
}

// GetCookiesDir 获取 cookies 目录
func (w *Wrapper) GetCookiesDir() string {
	return w.cookiesDir
}
