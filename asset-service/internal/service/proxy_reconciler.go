package service

import (
	"context"
	"log"
	"time"
)

type proxyBindingReconcileService interface {
	ReconcileProxyBindings(context.Context) (*ProxyBindingReconcileResult, error)
}

type ProxyBindingReconciler struct {
	service  proxyBindingReconcileService
	interval time.Duration
}

func NewProxyBindingReconciler(service proxyBindingReconcileService, interval time.Duration) *ProxyBindingReconciler {
	return &ProxyBindingReconciler{
		service:  service,
		interval: interval,
	}
}

func (r *ProxyBindingReconciler) Start(ctx context.Context) {
	if r == nil || r.service == nil || r.interval <= 0 {
		return
	}

	go r.loop(ctx)
}

func (r *ProxyBindingReconciler) loop(ctx context.Context) {
	r.run(ctx)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.run(ctx)
		}
	}
}

func (r *ProxyBindingReconciler) run(ctx context.Context) {
	result, err := r.service.ReconcileProxyBindings(ctx)
	if err != nil {
		log.Printf("[ProxyBindingReconciler] reconcile failed: %v", err)
		return
	}
	if result == nil || (result.TerminalBindingsReleased == 0 && result.ActiveTaskCountsUpdated == 0) {
		return
	}

	log.Printf(
		"[ProxyBindingReconciler] released_terminal_bindings=%d active_task_counts_updated=%d",
		result.TerminalBindingsReleased,
		result.ActiveTaskCountsUpdated,
	)
}
