import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
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
