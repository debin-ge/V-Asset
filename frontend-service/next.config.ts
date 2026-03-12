import type { NextConfig } from "next";

const appVersion =
  process.env.NEXT_PUBLIC_APP_VERSION ?? `build-${new Date().toISOString()}`;

const nextConfig: NextConfig = {
  output: "standalone",
  reactCompiler: true,
  env: {
    NEXT_PUBLIC_APP_VERSION: appVersion,
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "**.ytimg.com",
      },
      {
        protocol: "https",
        hostname: "**.youtube.com",
      },
      {
        protocol: "https",
        hostname: "**.hdslb.com",
      },
    ],
  },
};

export default nextConfig;
