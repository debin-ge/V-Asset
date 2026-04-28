-- Query indexes for proxy usage event observability.

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_platform_created_at
ON proxy_usage_events(platform, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_stage_created_at
ON proxy_usage_events(stage, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_success_created_at
ON proxy_usage_events(success, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_proxy_usage_events_category_created_at
ON proxy_usage_events(error_category, created_at DESC);
