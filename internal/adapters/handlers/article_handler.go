package handlers

import (
	"net/http"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/domain"
	"github.com/mbnaysinger/nayz-nexus-api/internal/core/services"
)

// ArticleHandler expõe as rotas públicas (sem auth): listagem e leitura
type ArticleHandler struct {
	articleService *services.ArticleService
}

func NewArticleHandler(articleService *services.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

type articleListResponse struct {
	Items []*domain.Article `json:"items"`
	Page  int               `json:"page"`
	Size  int               `json:"size"`
	Total int               `json:"total"`
}

// List — GET /api/v1/articles?page=1&size=10 (publicados, ordem cronológica, sem content_md)
func (h *ArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	size := queryInt(r, "size", 10)

	articles, total, err := h.articleService.ListPublished(r.Context(), page, size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleListResponse{Items: articles, Page: page, Size: size, Total: total})
}

// GetBySlug — GET /api/v1/articles/{slug} (artigo publicado completo)
func (h *ArticleHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	article, err := h.articleService.GetPublishedBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, article)
}
