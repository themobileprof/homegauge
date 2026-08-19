import type { NextConfig } from "next";

const apiProxy = process.env.API_PROXY_TARGET || "http://127.0.0.1:8080";
const basePath = process.env.NEXT_PUBLIC_BASE_PATH || "";

const nextConfig: NextConfig = {
  output: "standalone",
  basePath,
  env: {
    NEXT_PUBLIC_BASE_PATH: basePath,
  },
  // When deployed behind a reverse proxy with basePath, Next's image optimizer
  // cannot self-fetch /images/* correctly. Serve images as plain files via nginx.
  images: {
    unoptimized: !!basePath,
  },
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiProxy}/api/:path*` }];
  },
};

export default nextConfig;
