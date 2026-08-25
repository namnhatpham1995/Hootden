import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Lets `web/Dockerfile` copy only the traced production files, not the
  // full node_modules tree, into the final image.
  output: "standalone",
};

export default nextConfig;
