import type { NextConfig } from "next";

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
  // In dev, proxy /api/* to the local Go API.
  // In production, nginx routes /mortgage/api/* directly to the Go API — no baking needed.
  ...(!basePath && {
    async rewrites() {
      const apiProxy = process.env.API_PROXY_TARGET || "http://127.0.0.1:8080";
      return [{ source: "/api/:path*", destination: `${apiProxy}/api/:path*` }];
    },
  }),
};

export default nextConfig;
