import "server-only";

import fs from "node:fs";
import path from "node:path";

import type { PublicRuntimeConfig } from "./runtime-config";

const DEFAULT_PARSE_TIMEOUT_MS = 300000;
let cachedAppVersion = "";

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

function readFirstNonEmptyEnv(names: string[]): string {
  for (const name of names) {
    const value = readStringEnv(name);
    if (value) {
      return value;
    }
  }

  return "";
}

function readGitOrImageVersion(): string {
  return readFirstNonEmptyEnv([
    "APP_VERSION",
  ]);
}

function getTimestampCandidatePaths(): string[] {
  return [
    path.join(process.cwd(), ".next", "BUILD_ID"),
    path.join(process.cwd(), "BUILD_ID"),
    path.join(process.cwd(), "server.js"),
    path.join(process.cwd(), "frontend-service", ".next", "BUILD_ID"),
  ];
}

function formatBuildTimestamp(date: Date): string {
  return `build-${date.toISOString()}`;
}

function parseTimestamp(value: string): Date | null {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  return date;
}

function readBuildTimestamp(): string {
  const envTimestamp = readFirstNonEmptyEnv([
    "BUILD_TIMESTAMP",
  ]);
  if (envTimestamp) {
    const parsed = parseTimestamp(envTimestamp);
    if (parsed) {
      return formatBuildTimestamp(parsed);
    }
  }

  for (const candidate of getTimestampCandidatePaths()) {
    try {
      const stats = fs.statSync(candidate);
      if (stats.mtimeMs > 0) {
        return formatBuildTimestamp(stats.mtime);
      }
    } catch {
      // Ignore missing artifacts and continue to the next build timestamp candidate.
    }
  }

  return "";
}

export function getResolvedAppVersion(): string {
  if (cachedAppVersion) {
    return cachedAppVersion;
  }

  cachedAppVersion = readGitOrImageVersion() || readBuildTimestamp();
  return cachedAppVersion;
}

export function getPublicRuntimeConfig(): PublicRuntimeConfig {
  return {
    apiBaseUrl: readStringEnv("NEXT_PUBLIC_API_BASE_URL"),
    wsUrl: readStringEnv("NEXT_PUBLIC_WS_URL"),
    appVersion: getResolvedAppVersion(),
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
