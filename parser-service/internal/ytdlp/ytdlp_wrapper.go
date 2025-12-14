package ytdlp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"vasset/parser-service/internal/config"
	"vasset/parser-service/internal/utils"
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

// ExtractInfo 提取视频信息
func (w *Wrapper) ExtractInfo(url string, cookieFile string, extraArgs ...string) (*VideoInfo, error) {
	args := w.buildArgs(url, cookieFile, extraArgs)

	// 创建命令
	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.binaryPath, args...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 解析错误类型
		if ctx.Err() == context.DeadlineExceeded {
			return nil, utils.ErrTimeout
		}
		// 映射yt-dlp错误
		return nil, fmt.Errorf("%w: %s", utils.MapYTDLPError(string(output)), string(output))
	}

	// 解析JSON输出
	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	return &info, nil
}

// ExtractInfoWithProxy 使用指定代理提取视频信息 (临时覆盖默认代理)
func (w *Wrapper) ExtractInfoWithProxy(url, proxyURL, cookieFile string, extraArgs ...string) (*VideoInfo, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), w.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, w.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, utils.ErrTimeout
		}
		return nil, fmt.Errorf("%w: %s", utils.MapYTDLPError(string(output)), string(output))
	}

	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

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
