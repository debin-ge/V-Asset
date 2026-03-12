"use client";

import { useEffect } from "react";

const VERSION_CHECK_INTERVAL_MS = 5 * 60 * 1000;
const currentVersion = process.env.NEXT_PUBLIC_APP_VERSION ?? "unknown";

type VersionResponse = {
  version?: string;
};

function getReloadMarker(version: string) {
  return `reloaded-for-version:${version}`;
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
  return data.version ?? null;
}

export function VersionGuard() {
  useEffect(() => {
    if (process.env.NODE_ENV !== "production") {
      return;
    }

    let isChecking = false;

    const checkForUpdate = async () => {
      if (isChecking) {
        return;
      }

      isChecking = true;
      const controller = new AbortController();

      try {
        const latestVersion = await fetchLatestVersion(controller.signal);

        if (!latestVersion || latestVersion === currentVersion) {
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

    const intervalId = window.setInterval(() => {
      void checkForUpdate();
    }, VERSION_CHECK_INTERVAL_MS);

    void checkForUpdate();

    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("focus", handleFocus);

    return () => {
      window.clearInterval(intervalId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("focus", handleFocus);
    };
  }, []);

  return null;
}
