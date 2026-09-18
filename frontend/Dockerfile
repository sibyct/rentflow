# Production image: static build served by nginx.
FROM node:20-alpine AS build
WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm install

COPY . .

# Vite bakes VITE_* vars into the bundle at build time, not runtime —
# so the API URL has to come in as a build arg (set via `docker build
# --build-arg` / Fly's [build.args], see fly.toml/fly.staging.toml),
# not a container env var. No default: src/shared/lib/env.ts throws at
# app startup if this ends up empty, by design — better a loud, obvious
# error on first load than silent same-origin requests that 404.
ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf.template /etc/nginx/conf.d/default.conf.template

# Scheme + host only, no trailing slash, of the backend Fly app this
# frontend proxies /api/ to (see nginx.conf.template's comment for why
# a proxy rather than a direct cross-origin call). Set via Fly's
# [build.args] in fly.toml/fly.staging.toml.
ARG API_BACKEND_URL
# envsubst's variable list is deliberately restricted to just this one
# name — nginx's own directives use the same ${var}/$var syntax
# (${API_BACKEND_URL} vs $host, $uri, $scheme, ...), and an
# unrestricted envsubst would blank those out too, since they aren't
# real environment variables.
RUN apk add --no-cache gettext && \
    API_BACKEND_URL="$API_BACKEND_URL" envsubst '$API_BACKEND_URL' \
      < /etc/nginx/conf.d/default.conf.template > /etc/nginx/conf.d/default.conf && \
    rm /etc/nginx/conf.d/default.conf.template && \
    apk del gettext

EXPOSE 80
