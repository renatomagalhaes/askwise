# ============================================================================
# AskWise — Dockerfile Multi-Stage
# ============================================================================
# ADR-001: Docker-first — todo o build e execução acontecem dentro do container.
# Nenhuma instalação local de Go é necessária.
#
# Stages:
#   builder  — Compila os binários Go (server + chat)
#   runtime  — Imagem mínima para execução
#   dev      — Imagem com tooling para desenvolvimento (test, lint, etc.)
# ============================================================================

# ---------------------------------------------------------------------------
# Stage: builder — Compila os binários
# ---------------------------------------------------------------------------
FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache de dependências: copia go.mod/go.sum primeiro
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copia o código fonte
COPY . .

# Compila o servidor API
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/askwise-server ./cmd/server

# Compila o CLI de chat
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/askwise-chat ./cmd/chat

# ---------------------------------------------------------------------------
# Stage: runtime — Imagem mínima para execução
# ---------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates tzdata

# Cria usuário não-root
RUN adduser -D -g '' askwise
USER askwise

WORKDIR /app

# Copia os binários compilados
COPY --from=builder /bin/askwise-server /bin/askwise-server
COPY --from=builder /bin/askwise-chat /bin/askwise-chat

# Diretório para o SQLite
RUN mkdir -p /app/data

EXPOSE 8484

# Por padrão, roda o servidor API
CMD ["/bin/askwise-server"]

# ---------------------------------------------------------------------------
# Stage: dev — Imagem com Go completo para desenvolvimento
# ---------------------------------------------------------------------------
FROM golang:1.26.1-alpine AS dev

RUN apk add --no-cache git ca-certificates tzdata make curl jq

WORKDIR /app

# Cache de dependências
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

# Copia o código fonte (será sobrescrito pelo volume mount em dev)
COPY . .

# Porta do servidor
EXPOSE 8484

# Em modo dev, roda o servidor com go run (hot reload possível)
CMD ["go", "run", "./cmd/server"]
