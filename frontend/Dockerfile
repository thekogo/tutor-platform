# syntax=docker/dockerfile:1
# Build context: frontend/  (needs ../api/openapi.yaml only for `pnpm gen:api`,
# whose output src/lib/api/schema.d.ts is committed, so it is not needed here).
#
# Stage 1 builds the static export (out/); stage 2 serves it with nginx, which
# also proxies /api/* to BACKEND_URL (see nginx/default.conf.template).

ARG NODE_VERSION=22-alpine
ARG NGINX_VERSION=1.30-alpine

FROM node:${NODE_VERSION} AS base
ENV PNPM_HOME=/pnpm PATH=/pnpm:$PATH NEXT_TELEMETRY_DISABLED=1
RUN corepack enable
WORKDIR /app

FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile

FROM base AS build
# NEXT_PUBLIC_* are inlined at build time.
ARG NEXT_PUBLIC_DEV_LOGIN=false
ENV NEXT_PUBLIC_DEV_LOGIN=${NEXT_PUBLIC_DEV_LOGIN}
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN pnpm build

FROM nginx:${NGINX_VERSION} AS runtime
ENV PORT=3000 \
    BACKEND_URL=http://localhost:8080 \
    # Fill ${NGINX_LOCAL_RESOLVERS} from /etc/resolv.conf for the runtime resolver.
    NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1 \
    # Substitute only our variables, never nginx's own ($uri, $host, …).
    NGINX_ENVSUBST_FILTER="^(PORT|BACKEND_URL|NGINX_LOCAL_RESOLVERS)$"
# Our template replaces the stock server block; the IPv6 listen tweak would edit that one.
RUN rm -f /etc/nginx/conf.d/default.conf /docker-entrypoint.d/10-listen-on-ipv6-by-default.sh
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/out /usr/share/nginx/html
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=5 \
  CMD wget -q --spider "http://127.0.0.1:${PORT}/healthz" || exit 1
