package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	"github.com/mbnaysinger/nayz-nexus-api/internal/adapters/handlers"
	"github.com/mbnaysinger/nayz-nexus-api/internal/adapters/middlewares"
	"github.com/mbnaysinger/nayz-nexus-api/internal/adapters/repositories"
	"github.com/mbnaysinger/nayz-nexus-api/internal/adapters/storage"
	"github.com/mbnaysinger/nayz-nexus-api/internal/config"
	"github.com/mbnaysinger/nayz-nexus-api/internal/core/services"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
	slog.Info("Iniciando serviço nayz-nexus-api")

	godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")
	appID := os.Getenv("NAYZ_AUTH_APP_ID")

	if dsn == "" {
		slog.Error("Variável DATABASE_URL ausente")
		os.Exit(1)
	}
	if jwtSecret == "" {
		slog.Error("Variável JWT_SECRET ausente. Deve ser idêntica à do nayz-auth.")
		os.Exit(1)
	}
	if appID == "" {
		slog.Warn("NAYZ_AUTH_APP_ID não informado — tokens de qualquer aplicação do nayz-auth serão aceitos")
	}

	db := config.ConnectDB(dsn)
	defer db.Close()
	runMigrations(db)

	// Storage MinIO (instância já existente — local ou VPS). Sem credenciais o
	// serviço sobe mesmo assim: o presign responde 503 até configurar.
	mediaStorage := buildMinioStorage()

	// Repositories
	articleRepo := repositories.NewPostgresArticleRepository(db)

	// Services
	articleService := services.NewArticleService(articleRepo)
	mediaService := services.NewMediaService(mediaStorage)

	// Handlers
	articleHandler := handlers.NewArticleHandler(articleService)
	adminHandler := handlers.NewAdminArticleHandler(articleService)
	mediaHandler := handlers.NewMediaHandler(mediaService)

	authMiddleware := middlewares.NewAuthMiddleware(jwtSecret, appID)
	publicLimiter := middlewares.NewRateLimiter(envFloat("RATE_LIMIT_RPS", 5), envFloat("RATE_LIMIT_BURST", 20))
	mux := handlers.SetupRoutes(articleHandler, adminHandler, mediaHandler, authMiddleware, publicLimiter)

	corsOrigins := strings.Split(envOr("CORS_ORIGINS", "*"), ",")
	loggedRouter := middlewares.LoggerMiddleware(mux)
	corsRouter := middlewares.CorsMiddleware(corsOrigins, loggedRouter)

	httpPort := envOr("PORT", "8082")
	slog.Info("Servidor escutando chamadas", slog.String("port", httpPort))
	if err := http.ListenAndServe(":"+httpPort, corsRouter); err != nil {
		slog.Error("Falha crítica no servidor Web", "erro", err.Error())
		os.Exit(1)
	}
}

// buildMinioStorage retorna nil quando o MinIO ainda não foi configurado (envs vazias)
func buildMinioStorage() *storage.MinioStorage {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if endpoint == "" || accessKey == "" || secretKey == "" {
		slog.Warn("MinIO não configurado (MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY) — upload de mídia indisponível")
		return nil
	}

	useSSL := envOr("MINIO_USE_SSL", "false") == "true"

	// Tolerância de formato: o cliente S3 espera apenas host[:porta]. Se vier
	// URL completa (http(s)://host/caminho), extrai o host e infere o SSL.
	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" {
			slog.Error("MINIO_ENDPOINT inválido — use apenas host[:porta] da API S3", "valor", endpoint)
			return nil
		}
		useSSL = parsed.Scheme == "https"
		if parsed.Path != "" && parsed.Path != "/" {
			slog.Warn("MINIO_ENDPOINT tinha caminho — ignorado (a API S3 fica na raiz do host)",
				slog.String("informado", endpoint), slog.String("usado", parsed.Host))
		}
		endpoint = parsed.Host
	}
	endpoint = strings.TrimRight(endpoint, "/")

	bucket := envOr("MINIO_BUCKET", "nayztech-media")
	if strings.Contains(bucket, "/") {
		slog.Error("MINIO_BUCKET não pode conter '/' — use só o nome do bucket (o prefixo articles/ já é gerado nas chaves)",
			"valor", bucket)
		return nil
	}
	publicBase := envOr("MINIO_PUBLIC_URL", "http://"+endpoint+"/"+bucket)
	expiry := time.Duration(envFloat("PRESIGN_EXPIRY_MINUTES", 10)) * time.Minute

	mediaStorage, err := storage.NewMinioStorage(endpoint, accessKey, secretKey, bucket, publicBase, useSSL, expiry)
	if err != nil {
		slog.Error("Falha ao criar cliente MinIO — upload de mídia indisponível", "erro", err.Error())
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mediaStorage.EnsureBucket(ctx); err != nil {
		slog.Warn("Não foi possível garantir o bucket MinIO (verifique credenciais/rede)", "erro", err.Error())
	}
	return mediaStorage
}

func runMigrations(db *sqlx.DB) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		slog.Error("Não instanciou driver migração", "erro", err.Error())
		os.Exit(1)
	}
	m, err := migrate.NewWithDatabaseInstance("file://db/migrations", "postgres", driver)
	if err != nil {
		slog.Error("Não inicializou migração", "erro", err.Error())
		os.Exit(1)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("Erro crítico nas migrações", "erro", err.Error())
		os.Exit(1)
	}
	slog.Info("Migrações de banco de dados atualizadas com sucesso!")
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func envFloat(name string, fallback float64) float64 {
	if raw := os.Getenv(name); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil && v > 0 {
			return v
		}
	}
	return fallback
}
