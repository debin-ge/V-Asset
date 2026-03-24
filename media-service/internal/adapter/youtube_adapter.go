package adapter

import (
	"context"
	"log"
	"strings"

	"youdlp/media-service/internal/utils"
	"youdlp/media-service/internal/ytdlp"
)

// YouTubeAdapter YouTube平台适配器
type YouTubeAdapter struct {
	ytdlp      *ytdlp.Wrapper
	cookieFile string
	args       []string
}

// NewYouTubeAdapter 创建YouTube适配器
func NewYouTubeAdapter(wrapper *ytdlp.Wrapper, cookieFile string, extraArgs []string) *YouTubeAdapter {
	return &YouTubeAdapter{
		ytdlp:      wrapper,
		cookieFile: cookieFile,
		args:       extraArgs,
	}
}

// Parse 解析YouTube视频（使用静态 cookie）
func (a *YouTubeAdapter) Parse(ctx context.Context, url string) (*ytdlp.VideoInfo, error) {
	return a.ytdlp.ExtractInfo(ctx, url, a.cookieFile, a.args...)
}

// ParseWithCookie 解析YouTube视频（使用动态 cookie）
func (a *YouTubeAdapter) ParseWithCookie(ctx context.Context, url string, cookieFile string) (*ytdlp.VideoInfo, error) {
	// 优先使用传入的 cookie，如果为空则使用静态 cookie
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}
	return a.ytdlp.ExtractInfo(ctx, url, cookie, a.args...)
}

// ParseWithProxyAndCookie 解析YouTube视频（使用动态 proxy 和 cookie）
func (a *YouTubeAdapter) ParseWithProxyAndCookie(ctx context.Context, url, proxyURL, cookieFile string) (*ytdlp.VideoInfo, error) {
	// 优先使用传入的 cookie，如果为空则使用静态 cookie
	cookie := cookieFile
	if cookie == "" {
		cookie = a.cookieFile
	}

	info, err := a.ytdlp.ExtractInfoWithProxy(ctx, url, proxyURL, cookie, a.args...)
	if err != nil {
		return nil, err
	}

	if hasSufficientYouTubeFormats(info) {
		return info, nil
	}

	log.Printf("[YT-DLP-PARSE] YouTube parse returned limited formats (%d); retrying with broader player clients", len(info.Formats))
	broaderArgs := broadenYouTubePlayerClients(a.args)
	if !argsChanged(a.args, broaderArgs) {
		return info, nil
	}

	broaderInfo, broaderErr := a.ytdlp.ExtractInfoWithProxy(ctx, url, proxyURL, cookie, broaderArgs...)
	if broaderErr != nil {
		log.Printf("[YT-DLP-PARSE] Broader player-client retry failed: %v", broaderErr)
		return info, nil
	}

	mergedInfo := mergeVideoInfos(info, broaderInfo)
	if hasSufficientYouTubeFormats(mergedInfo) || len(mergedInfo.Formats) > len(info.Formats) {
		log.Printf("[YT-DLP-PARSE] Broader player clients increased available formats from %d to %d", len(info.Formats), len(mergedInfo.Formats))
		return mergedInfo, nil
	}

	return info, nil
}

func hasSufficientYouTubeFormats(info *ytdlp.VideoInfo) bool {
	if info == nil {
		return false
	}

	videoHeights := make(map[int]struct{})
	videoFormats := 0
	audioOnlyFormats := 0

	for _, format := range info.Formats {
		hasVideo := format.VCodec != "" && format.VCodec != "none"
		hasAudio := format.ACodec != "" && format.ACodec != "none"

		if hasVideo {
			videoFormats++
			if format.Height > 0 {
				videoHeights[format.Height] = struct{}{}
			}
		}
		if hasAudio && !hasVideo {
			audioOnlyFormats++
		}
	}

	return len(videoHeights) > 1 || (videoFormats > 1 && audioOnlyFormats > 0)
}

func broadenYouTubePlayerClients(args []string) []string {
	updated := make([]string, 0, len(args))
	replaced := false

	for i := 0; i < len(args); i++ {
		if args[i] == "--extractor-args" && i+1 < len(args) {
			value := args[i+1]
			if strings.HasPrefix(value, "youtube:player_client=") {
				updated = append(updated, "--extractor-args", "youtube:player_client=web,android,tv,web_safari")
				replaced = true
				i++
				continue
			}
		}
		updated = append(updated, args[i])
	}

	if !replaced {
		updated = append(updated, "--extractor-args", "youtube:player_client=web,android,tv,web_safari")
	}

	return updated
}

func argsChanged(original, updated []string) bool {
	if len(original) != len(updated) {
		return true
	}
	for i := range original {
		if original[i] != updated[i] {
			return true
		}
	}
	return false
}

func mergeVideoInfos(primary, secondary *ytdlp.VideoInfo) *ytdlp.VideoInfo {
	if primary == nil {
		return secondary
	}
	if secondary == nil {
		return primary
	}

	merged := *primary
	merged.Formats = append([]utils.VideoFormat(nil), primary.Formats...)

	seen := make(map[string]struct{}, len(primary.Formats))
	for _, format := range primary.Formats {
		seen[formatFingerprint(format)] = struct{}{}
	}

	for _, format := range secondary.Formats {
		key := formatFingerprint(format)
		if _, ok := seen[key]; ok {
			continue
		}
		merged.Formats = append(merged.Formats, format)
		seen[key] = struct{}{}
	}

	return &merged
}

func formatFingerprint(format utils.VideoFormat) string {
	return strings.Join([]string{
		format.FormatID,
		format.Ext,
		format.VCodec,
		format.ACodec,
		strings.TrimSpace(format.Resolution),
	}, "|")
}
