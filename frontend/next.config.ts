import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  allowedDevOrigins: ["106.15.3.98"],
  output: "standalone", // 生产部署优化
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*",
      },
    ];
  },
};

export default nextConfig;
