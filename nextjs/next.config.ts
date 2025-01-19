import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  experimental: {
    reactCompiler: true,
    serverActions: {
      allowedOrigins: ["localhost", "localhost:8080"],
    },
  },
  images: {
    remotePatterns: [
      {
        protocol: "https",
        hostname: "h3.googleusercontent.com",
      },
    ],
  },
};

export default nextConfig;
