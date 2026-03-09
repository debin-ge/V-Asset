import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  output: "standalone",
  basePath: "/admin-console",
  turbopack: {
    root: path.join(__dirname),
  },
};

export default nextConfig;
