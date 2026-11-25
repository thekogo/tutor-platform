import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

/**
 * `next build` emits a static site into out/ (output: "export"), served by
 * nginx (see Dockerfile and nginx/), which also proxies /api/* to the Go
 * backend and redirects legacy URLs.
 *
 * `next dev` has no nginx in front, so the dev server proxies /api/* to
 * BACKEND_URL itself (rewrites are dev-only: a static export cannot use them).
 */
export default function config(phase: string): NextConfig {
  const isDev = phase === PHASE_DEVELOPMENT_SERVER;
  const backend = (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/+$/, "");
  return {
    ...(isDev
      ? { rewrites: async () => [{ source: "/api/:path*", destination: `${backend}/api/:path*` }] }
      : { output: "export" }),
    reactStrictMode: true,
    poweredByHeader: false,
    images: { unoptimized: true },
    turbopack: {
      rules: {
        "*.css": {
          loaders: ["@tailwindcss/turbopack"],
          as: "*.css",
        },
      },
    },
  };
}
