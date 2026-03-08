import type { NextConfig } from "next";
import path from "path";

const apiGatewayOrigin = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  turbopack: {
    root: path.join(__dirname),
  },
  async rewrites() {
    return [
      {
        source: "/api/v1/admin/:path*",
        destination: `${apiGatewayOrigin}/api/v1/admin/:path*`,
      },
    ];
  },
};

export default nextConfig;
