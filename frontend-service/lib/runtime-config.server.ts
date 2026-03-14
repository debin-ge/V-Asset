import "server-only";

import type { PublicRuntimeConfig } from "./runtime-config";

const DEFAULT_APP_VERSION = "unknown";
const DEFAULT_PARSE_TIMEOUT_MS = 300000;

function readStringEnv(name: string): string {
  return process.env[name]?.trim() ?? "";
}

function readNumberEnv(name: string, fallback: number): number {
  const value = process.env[name];
  if (!value) {
    return fallback;
  }

  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

export function getPublicRuntimeConfig(): PublicRuntimeConfig {
  return {
    apiBaseUrl: readStringEnv("NEXT_PUBLIC_API_BASE_URL"),
    wsUrl: readStringEnv("NEXT_PUBLIC_WS_URL"),
    appVersion: readStringEnv("NEXT_PUBLIC_APP_VERSION") || DEFAULT_APP_VERSION,
    parseTimeoutMs: readNumberEnv(
      "NEXT_PUBLIC_PARSE_TIMEOUT_MS",
      DEFAULT_PARSE_TIMEOUT_MS
    ),
  };
}

export function serializePublicRuntimeConfig(
  config: PublicRuntimeConfig
): string {
  const serialized = JSON.stringify(config).replace(/</g, "\\u003c");
  return `window.__V_ASSET_RUNTIME_CONFIG__=${serialized};`;
}
