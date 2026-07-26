package handlers

import (
	"net/http"

	"github.com/mbnaysinger/nayz-nexus-api/internal/adapters/middlewares"
)

func SetupRoutes(
	articleHandler *ArticleHandler,
	adminHandler *AdminArticleHandler,
	mediaHandler *MediaHandler,
	authMiddleware *middlewares.AuthMiddleware,
	publicLimiter *middlewares.RateLimiter,
) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "UP", "message": "Serviço nayz-nexus-api operando normalmente"}`))
	})

	requireAdmin := func(next http.HandlerFunc) http.HandlerFunc {
		return authMiddleware.RequireRole(middlewares.RoleAdmin, next)
	}
	limited := publicLimiter.Limit

	// ---- Público (rate-limit básico, spec §6) ----
	mux.HandleFunc("GET /api/v1/articles", limited(articleHandler.List))
	mux.HandleFunc("GET /api/v1/articles/{slug}", limited(articleHandler.GetBySlug))

	// ---- Admin (Bearer JWT nayz-auth, role ADMIN da app NAYZTECH) ----
	mux.HandleFunc("GET /api/v1/admin/articles", requireAdmin(adminHandler.List))
	mux.HandleFunc("POST /api/v1/admin/articles", requireAdmin(adminHandler.Create))
	mux.HandleFunc("GET /api/v1/admin/articles/{id}", requireAdmin(adminHandler.Get))
	mux.HandleFunc("PUT /api/v1/admin/articles/{id}", requireAdmin(adminHandler.Update))
	mux.HandleFunc("DELETE /api/v1/admin/articles/{id}", requireAdmin(adminHandler.Delete))
	mux.HandleFunc("PATCH /api/v1/admin/articles/{id}/publish", requireAdmin(adminHandler.Publish))
	mux.HandleFunc("PATCH /api/v1/admin/articles/{id}/unpublish", requireAdmin(adminHandler.Unpublish))

	// ---- Mídia (presigned upload direto para o MinIO) ----
	mux.HandleFunc("POST /api/v1/admin/media/presign", requireAdmin(mediaHandler.Presign))

	return mux
}
