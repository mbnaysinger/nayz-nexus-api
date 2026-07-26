package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/services"
)

// MediaHandler emite presigned URLs de upload para o MinIO (admin)
type MediaHandler struct {
	mediaService *services.MediaService
}

func NewMediaHandler(mediaService *services.MediaService) *MediaHandler {
	return &MediaHandler{mediaService: mediaService}
}

type presignRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// Presign — POST /api/v1/admin/media/presign { filename, content_type }
func (h *MediaHandler) Presign(w http.ResponseWriter, r *http.Request) {
	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Formato de JSON inválido")
		return
	}

	upload, err := h.mediaService.Presign(r.Context(), req.Filename, req.ContentType)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, upload)
}
