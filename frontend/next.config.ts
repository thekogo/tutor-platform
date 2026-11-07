import type { NextConfig } from "next";

// /api/* is forwarded to the Go backend at request time by src/proxy.ts
// (so BACKEND_URL is a runtime env var, not baked in at build time).
// In mock mode (NEXT_PUBLIC_API_MOCKING=enabled) the catch-all route handler
// at src/app/api/[...path]/route.ts serves the MSW handlers instead.
const nextConfig: NextConfig = {
  output: "standalone",
  reactStrictMode: true,
  poweredByHeader: false,
  experimental: {
    // forbidden() → app/forbidden.tsx with a real 403 status (non-Owner on /admin).
    authInterrupts: true,
  },
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
