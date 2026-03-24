import Script from "next/script";

export function RuntimeConfigScript() {
  return (
    <Script
      id="youdlp-runtime-config"
      src="/runtime-config"
      strategy="afterInteractive"
    />
  );
}
