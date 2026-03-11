package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	downloadconfig "vasset/media-service/internal/download/config"
	downloadmodels "vasset/media-service/internal/download/models"
)

func TestDownloadConcurrentCallbacksRemainIsolated(t *testing.T) {
	t.Parallel()

	binaryPath := writeFakeDownloader(t)
	executor := NewExecutor(&downloadconfig.YtDLPConfig{
		BinaryPath:          binaryPath,
		Timeout:             5,
		ConcurrentFragments: 1,
	})

	ctx := context.Background()
	taskA := &downloadmodels.DownloadTask{TaskID: "task-a", URL: "https://example.com/a", Format: "mp4"}
	taskB := &downloadmodels.DownloadTask{TaskID: "task-b", URL: "https://example.com/b", Format: "mp4"}

	var mu sync.Mutex
	callbacks := map[string][]float64{
		"task-a": {},
		"task-b": {},
	}

	record := func(taskID string) func(*OutputEvent) {
		return func(event *OutputEvent) {
			if event.Type != "progress" || event.Progress == nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			callbacks[taskID] = append(callbacks[taskID], event.Progress.Percent)
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := executor.Download(ctx, taskA, "", filepath.Join(t.TempDir(), "a.mp4"), "", record("task-a")); err != nil {
			t.Errorf("task-a download failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := executor.Download(ctx, taskB, "", filepath.Join(t.TempDir(), "b.mp4"), "", record("task-b")); err != nil {
			t.Errorf("task-b download failed: %v", err)
		}
	}()

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	assertProgressSequence(t, callbacks["task-a"], []float64{10, 50, 100}, "task-a")
	assertProgressSequence(t, callbacks["task-b"], []float64{10, 50, 100}, "task-b")
}

func TestBuildCommandUsesExactSelectedFormat(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(&downloadconfig.YtDLPConfig{
		BinaryPath:          "/usr/local/bin/yt-dlp",
		Timeout:             5,
		ConcurrentFragments: 1,
	})

	task := &downloadmodels.DownloadTask{
		TaskID:   "task-selected",
		URL:      "https://example.com/video",
		Format:   "webm",
		FormatID: "248",
		SelectedFormat: &downloadmodels.SelectedFormat{
			FormatID:   "248",
			Extension:  "webm",
			VideoCodec: "vp09",
			AudioCodec: "none",
		},
	}

	cmd := executor.buildCommand(task, "", "/tmp/output.webm", "")
	args := strings.Join(cmd.Args, " ")

	if !strings.Contains(args, "--format 248+bestaudio[ext=webm]/248+bestaudio/248/best") {
		t.Fatalf("expected exact selected format to be used, got %q", args)
	}
	if !strings.Contains(args, "--merge-output-format webm") {
		t.Fatalf("expected merge output format to match selected extension, got %q", args)
	}
	if !strings.Contains(args, "--print before_dl:[format-trace] format_id=%(format_id)s") {
		t.Fatalf("expected before_dl trace print to be configured, got %q", args)
	}
	if !strings.Contains(args, "--print after_move:[file-trace] filepath=%(filepath)s") {
		t.Fatalf("expected after_move trace print to be configured, got %q", args)
	}
}

func TestBuildCommandSkipsMergeForExactAudioFormat(t *testing.T) {
	t.Parallel()

	executor := NewExecutor(&downloadconfig.YtDLPConfig{
		BinaryPath:          "/usr/local/bin/yt-dlp",
		Timeout:             5,
		ConcurrentFragments: 1,
	})

	task := &downloadmodels.DownloadTask{
		TaskID:   "task-audio",
		URL:      "https://example.com/audio",
		Format:   "m4a",
		FormatID: "140",
		SelectedFormat: &downloadmodels.SelectedFormat{
			FormatID:   "140",
			Extension:  "m4a",
			VideoCodec: "none",
			AudioCodec: "mp4a.40.2",
		},
	}

	cmd := executor.buildCommand(task, "", "/tmp/output.m4a", "")
	args := strings.Join(cmd.Args, " ")

	if strings.Contains(args, "--merge-output-format") {
		t.Fatalf("did not expect merge output format for exact audio download, got %q", args)
	}
	if !strings.Contains(args, "--format 140") {
		t.Fatalf("expected exact audio format to be used, got %q", args)
	}
}

func assertProgressSequence(t *testing.T, got, want []float64, taskID string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s expected %d callbacks, got %d (%v)", taskID, len(want), len(got), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s expected callback %d to be %.1f, got %.1f (all=%v)", taskID, i, want[i], got[i], got)
		}
	}
}

func writeFakeDownloader(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fake-ytdlp.sh")
	script := "#!/bin/sh\n" +
		"echo '[download]  10.0% of 100.00MiB at 2.50MiB/s ETA 00:22'\n" +
		"sleep 0.05\n" +
		"echo '[download]  50.0% of 100.00MiB at 2.50MiB/s ETA 00:10'\n" +
		"sleep 0.05\n" +
		"echo '[download]  100.0% of 100.00MiB at 2.50MiB/s ETA 00:00'\n"

	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake downloader: %v", err)
	}

	// 部分环境对 shebang 解析较严格，显式等待权限落盘。
	time.Sleep(10 * time.Millisecond)
	return path
}
