FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/server ./main.go

FROM node:22-alpine
RUN apk add --no-cache ca-certificates tzdata git openssh-client docker-cli curl

# Install CLI agents
RUN npm install -g @anthropic-ai/claude-code @openai/codex

# Install pnpm (JS package manager) and uv (Python package manager)
RUN npm install -g pnpm && \
    curl -LsSf https://astral.sh/uv/install.sh | sh && \
    ln -s /root/.local/bin/uv /usr/local/bin/uv && \
    ln -s /root/.local/bin/uvx /usr/local/bin/uvx

COPY --from=builder /app/server /server
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/stack_registry /stack_registry

# Workspace directories are mounted at runtime via docker-compose bind mounts.
# Each worker's workspace is mounted at /workspaces/<worker-name>.
RUN mkdir -p /workspaces /app/infra/nginx

EXPOSE 8080
ENTRYPOINT ["/server"]
