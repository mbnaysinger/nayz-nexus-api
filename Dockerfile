# ---------- Build ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache de dependências
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Binário estático (sem CGO) para rodar em imagem mínima
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /nayz-nexus-api ./cmd/nayz-nexus-api

# ---------- Runtime ----------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /nayz-nexus-api ./nayz-nexus-api
# As migrations rodam automaticamente no start (golang-migrate, source file://db/migrations)
COPY --from=builder /app/db/migrations ./db/migrations

USER app

# Configuração via ambiente (ver .env.example / README):
# DATABASE_URL e JWT_SECRET obrigatórias | NAYZ_AUTH_APP_ID | CORS_ORIGINS
# MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET/PUBLIC_URL | PORT (default 8082)
EXPOSE 8082

ENTRYPOINT ["./nayz-nexus-api"]
