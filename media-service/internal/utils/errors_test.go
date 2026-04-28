package utils

import (
	"fmt"
	"testing"
)

func TestIsProxyOrBotRetryableErrorDetectsYouTubeBotMessage(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("%w: Sign in to confirm you’re not a bot. Use --cookies-from-browser or --cookies for the authentication.", ErrYTDLPFailed)
	if !IsProxyOrBotRetryableError(err) {
		t.Fatalf("expected YouTube bot-detection error to be retryable")
	}
}

func TestIsProxyOrBotRetryableErrorSkipsTerminalVideoErrors(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("%w: private video", ErrVideoPrivate)
	if IsProxyOrBotRetryableError(err) {
		t.Fatalf("expected terminal video error to be non-retryable")
	}
}
