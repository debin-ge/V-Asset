export interface PublicRuntimeConfig {
  apiBaseUrl: string;
  wsUrl: string;
  appVersion: string;
  parseTimeoutMs: number;
}

declare global {
  interface Window {
    __V_ASSET_RUNTIME_CONFIG__?: Partial<PublicRuntimeConfig>;
  }
}

const DEFAULT_APP_VERSION = "unknown";
const DEFAULT_PARSE_TIMEOUT_MS = 300000;

function readWindowRuntimeConfig(): Partial<PublicRuntimeConfig> {
  if (typeof window === "undefined") {
    return {};
  }

  return window.__V_ASSET_RUNTIME_CONFIG__ ?? {};
}

function normalizeString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeNumber(value: unknown, fallback: number): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed) && parsed > 0) {
      return parsed;
    }
  }

  return fallback;
}

export function getRuntimeConfig(): PublicRuntimeConfig {
  const config = readWindowRuntimeConfig();

  return {
    apiBaseUrl: normalizeString(config.apiBaseUrl),
    wsUrl: normalizeString(config.wsUrl),
    appVersion: normalizeString(config.appVersion) || DEFAULT_APP_VERSION,
    parseTimeoutMs: normalizeNumber(
      config.parseTimeoutMs,
      DEFAULT_PARSE_TIMEOUT_MS
    ),
  };
}

export function resolveApiBaseUrl(): string {
  const explicitBaseUrl = getRuntimeConfig().apiBaseUrl;
  if (explicitBaseUrl) {
    return explicitBaseUrl;
  }

  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  return "http://localhost:8080";
}

export function resolveWsBaseUrl(): string {
  const { wsUrl, apiBaseUrl } = getRuntimeConfig();
  if (wsUrl) {
    return wsUrl;
  }

  if (apiBaseUrl) {
    try {
      const apiUrl = new URL(apiBaseUrl);
      apiUrl.protocol = apiUrl.protocol === "https:" ? "wss:" : "ws:";
      return apiUrl.origin;
    } catch (error) {
      console.warn(
        "Invalid NEXT_PUBLIC_API_BASE_URL for WebSocket resolution:",
        error
      );
    }
  }

  if (typeof window !== "undefined") {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${protocol}//${window.location.host}`;
  }

  return "ws://localhost:8080";
}

export function resolveParseTimeoutMs(): number {
  return getRuntimeConfig().parseTimeoutMs;
}

export function resolveAppVersion(): string {
  return getRuntimeConfig().appVersion;
}
