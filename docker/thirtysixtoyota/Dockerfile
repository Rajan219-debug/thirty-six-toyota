# Thirty Six Toyota — self-contained build of THIS fork (custom accountability +
# timesheets/reporting), not the upstream vikunja/vikunja image. Frontend is
# embedded into the Go binary (frontend/embed.go), so the result is one binary
# that serves both the API and the web UI.
#
# Build context is the repo root (see docker-compose.yml). Lighter than the
# official xgo release Dockerfile: plain `go build` for the local platform.

# 1. Build the frontend (dist/ is embedded by the Go build below).
FROM node:24-alpine AS frontend
WORKDIR /build
ENV PUPPETEER_SKIP_DOWNLOAD=true
ENV CYPRESS_INSTALL_BINARY=0
COPY frontend/pnpm-lock.yaml frontend/package.json frontend/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

# 2. Build the API, embedding the freshly built frontend dist.
FROM golang:1.26-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The root .dockerignore drops the host's frontend/dist; use the built one.
COPY --from=frontend /build/dist ./frontend/dist
# -buildvcs=false: .git is excluded from the build context, so don't stamp VCS.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -o /vikunja .

# 3. Minimal runtime.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1000 vikunja && \
    mkdir -p /app/vikunja/files && chown -R vikunja:vikunja /app/vikunja
COPY --from=api /vikunja /usr/local/bin/vikunja
WORKDIR /app/vikunja
USER vikunja
ENV VIKUNJA_SERVICE_ROOTPATH=/app/vikunja/
EXPOSE 3456
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=5 \
    CMD wget -qO- http://localhost:3456/api/v1/info >/dev/null 2>&1 || exit 1
ENTRYPOINT ["vikunja"]
CMD ["web"]
