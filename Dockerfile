# syntax=docker/dockerfile:1

# ── Frontend: build the SPA that gets embedded into the binary ─────────
FROM node:24.18.0-alpine AS frontend
WORKDIR /build
ENV PUPPETEER_SKIP_DOWNLOAD=true \
    CYPRESS_INSTALL_BINARY=0
COPY frontend/pnpm-lock.yaml frontend/package.json frontend/pnpm-workspace.yaml ./
RUN npm install -g corepack && corepack enable && pnpm install --frozen-lockfile
COPY frontend/ ./
ARG RELEASE_VERSION=dev
RUN echo "{\"VERSION\": \"${RELEASE_VERSION}\"}" > src/version.json && pnpm run build

# ── Backend: embed dist and build one static, CGO-free binary ─────────
# Postgres is the deployment target (see docker-compose.yml), so no sqlite/
# cgo is needed — a static build keeps the runtime image tiny and portable.
FROM golang:1.26 AS backend
WORKDIR /src
COPY go.mod go.sum ./
# Cache mounts keep modules and build cache across builds, so a flaky module
# download doesn't force a full re-fetch on retry.
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . ./
COPY --from=frontend /build/dist ./frontend/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vikunja .

# ── Runtime: minimal, non-root ────────────────────────────────────────
# TLS certs and timezone data are copied from the build stage rather than
# installed via apk, so the runtime layer needs no network. busybox (bundled
# with alpine) provides adduser and the wget used by the healthcheck.
FROM alpine:3.20
LABEL org.opencontainers.image.title="Thirty Six Toyota"
LABEL org.opencontainers.image.description="Task accountability platform for Thirty Six Toyota dealerships"
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=backend /usr/share/zoneinfo /usr/share/zoneinfo
RUN adduser -D -u 1000 vikunja
WORKDIR /app/vikunja
COPY --from=backend /vikunja /app/vikunja/vikunja
# Pre-create the files dir owned by the app user: Docker seeds a fresh named
# volume (docker-compose.yml's vikunja-files) from whatever already exists in
# the image at its mount point, ownership included. Without this, the volume
# gets created owned by root and the non-root process below can't write to it.
RUN mkdir -p /app/vikunja/files && chown -R vikunja:vikunja /app/vikunja/files
USER vikunja
ENV VIKUNJA_SERVICE_ROOTPATH=/app/vikunja/
EXPOSE 3456
HEALTHCHECK --interval=30s --timeout=5s --start-period=25s \
    CMD wget -qO- http://localhost:3456/api/v1/info || exit 1
ENTRYPOINT ["/app/vikunja/vikunja"]
