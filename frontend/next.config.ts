import type { NextConfig } from "next";

const apiProxy = process.env.API_PROXY_TARGET || "http://127.0.0.1:8080";

const nextConfig: NextConfig = {
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiProxy}/api/:path*` }];
  },
};

export default nextConfig;
