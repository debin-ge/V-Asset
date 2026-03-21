package ytdlp

import "testing"

func TestParseDownloadProgressParsesCompleteLine(t *testing.T) {
	progress := parseDownloadProgress("[download]  45.2% of 100.00MiB at 2.50MiB/s ETA 00:22")
	if progress == nil {
		t.Fatal("expected progress to be parsed")
	}
	if progress.Percent != 45.2 {
		t.Fatalf("unexpected percent: %v", progress.Percent)
	}
	if progress.TotalBytes != 104857600 {
		t.Fatalf("unexpected total bytes: %d", progress.TotalBytes)
	}
	if progress.Speed != "2.50MiB/s" {
		t.Fatalf("unexpected speed: %q", progress.Speed)
	}
	if progress.ETA != "00:22" {
		t.Fatalf("unexpected eta: %q", progress.ETA)
	}
}

func TestParseDownloadProgressRejectsNonDownloadLine(t *testing.T) {
	if progress := parseDownloadProgress("[info] processing metadata"); progress != nil {
		t.Fatal("expected non-download line to be ignored")
	}
}
