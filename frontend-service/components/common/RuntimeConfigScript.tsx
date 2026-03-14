import Script from "next/script";

export function RuntimeConfigScript() {
  return (
    <Script
      id="vasset-runtime-config"
      src="/runtime-config"
      strategy="afterInteractive"
    />
  );
}
