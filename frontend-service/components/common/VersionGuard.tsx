"use client";

import { useEffect } from "react";

const MIN_CHECK_GAP_MS = 30 * 1000;

type VersionResponse = {
  version?: string;
};

type VersionGuardProps = {
  version: string;
};

function getReloadMarker(version: string) {
  return `reloaded-for-version:${version}`;
}

function normalizeVersion(value: string | null | undefined) {
  return typeof value === "string" ? value.trim() : "";
}

async function fetchLatestVersion(signal?: AbortSignal) {
  const response = await fetch("/app-version", {
    cache: "no-store",
    signal,
  });

  if (!response.ok) {
    return null;
  }

  const data = (await response.json()) as VersionResponse;
  return normalizeVersion(data.version) || null;
}

export function VersionGuard({ version }: VersionGuardProps) {
  useEffect(() => {
    if (process.env.NODE_ENV !== "production") {
      return;
    }

    let currentVersion = normalizeVersion(version);
    let isChecking = false;
    let lastCheckAt = 0;

    const checkForUpdate = async (force = false) => {
      const now = Date.now();
      if (isChecking) {
        return;
      }
      if (!force && now-lastCheckAt < MIN_CHECK_GAP_MS) {
        return;
      }

      isChecking = true;
      lastCheckAt = now;
      const controller = new AbortController();

      try {
        const latestVersion = await fetchLatestVersion(controller.signal);
        if (!latestVersion) {
          return;
        }

        if (!currentVersion) {
          currentVersion = latestVersion;
          return;
        }

        if (latestVersion === currentVersion) {
          return;
        }

        const marker = getReloadMarker(latestVersion);
        if (window.sessionStorage.getItem(marker)) {
          return;
        }

        window.sessionStorage.setItem(marker, "true");
        window.location.reload();
      } catch {
        // Ignore transient version check failures to avoid disrupting the user.
      } finally {
        controller.abort();
        isChecking = false;
      }
    };

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        void checkForUpdate();
      }
    };

    const handleFocus = () => {
      void checkForUpdate();
    };

    void checkForUpdate(true);

    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("focus", handleFocus);

    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("focus", handleFocus);
    };
  }, [version]);

  return null;
}
