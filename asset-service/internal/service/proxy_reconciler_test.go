package service

import (
	"context"
	"testing"
	"time"
)

type fakeProxyBindingReconcileService struct {
	calls chan struct{}
}

func (f *fakeProxyBindingReconcileService) ReconcileProxyBindings(context.Context) (*ProxyBindingReconcileResult, error) {
	f.calls <- struct{}{}
	return &ProxyBindingReconcileResult{}, nil
}

func TestProxyBindingReconcilerRunsImmediately(t *testing.T) {
	t.Parallel()

	fake := &fakeProxyBindingReconcileService{calls: make(chan struct{}, 1)}
	reconciler := NewProxyBindingReconciler(fake, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reconciler.Start(ctx)

	select {
	case <-fake.calls:
	case <-time.After(time.Second):
		t.Fatal("expected reconciler to run immediately after start")
	}
}
