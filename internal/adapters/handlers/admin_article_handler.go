package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/services"
)

// AdminArticleHandler expõe o CRUD protegido (role ADMIN da app NAYZTECH)
type AdminArticleHandler struct {
	articleService *services.ArticleService
}

func NewAdminArticleHandler(articleService *services.ArticleService) *AdminArticleHandler {
	return &AdminArticleHandler{articleService: articleService}
}

// List — GET /api/v1/admin/articles (todos, inclusive drafts)
func (h *AdminArticleHandler) List(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	size := queryInt(r, "size", 50)

	articles, total, err := h.articleService.ListAll(r.Context(), page, size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, articleListResponse{Items: articles, Page: page, Size: size, Total: total})
}

// Get — GET /api/v1/admin/articles/{id} (completo, para o editor)
func (h *AdminArticleHandler) Get(w http.ResponseWriter, r *http.Request) {
	article, err := h.articleService.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, article)
}

// Create — POST /api/v1/admin/articles (nasce como draft)
func (h *AdminArticleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input services.ArticleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Formato de JSON inválido")
		return
	}

	article, err := h.articleService.Create(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, article)
}

// Update — PUT /api/v1/admin/articles/{id}
func (h *AdminArticleHandler) Update(w http.ResponseWriter, r *http.Request) {
	var input services.ArticleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Formato de JSON inválido")
		return
	}

	article, err := h.articleService.Update(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, article)
}

// Delete — DELETE /api/v1/admin/articles/{id}
func (h *AdminArticleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.articleService.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Publish — PATCH /api/v1/admin/articles/{id}/publish
func (h *AdminArticleHandler) Publish(w http.ResponseWriter, r *http.Request) {
	article, err := h.articleService.Publish(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, article)
}

// Unpublish — PATCH /api/v1/admin/articles/{id}/unpublish
func (h *AdminArticleHandler) Unpublish(w http.ResponseWriter, r *http.Request) {
	article, err := h.articleService.Unpublish(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, article)
}
