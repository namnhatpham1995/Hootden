import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Lets `web/Dockerfile` copy only the traced production files, not the
  // full node_modules tree, into the final image. Vercel has its own build
  // output and breaks (every route 404s) if this is forced on, so only set
  // it for the Docker build -- Vercel sets VERCEL=1 during its own builds.
  output: process.env.VERCEL ? undefined : "standalone",

  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${process.env.API_PROXY_TARGET ?? "http://localhost:8080"}/:path*`,
      },
    ];
  },
};

export default nextConfig;
