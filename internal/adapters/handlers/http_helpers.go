package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/services"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeServiceError mapeia erros de negócio para status HTTP sem vazar detalhes internos
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidArticle), errors.Is(err, services.ErrInvalidMedia):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrArticleNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, services.ErrSlugConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, services.ErrStorageUnavailable):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	default:
		slog.Error("erro interno no handler", "erro", err.Error())
		writeError(w, http.StatusInternalServerError, "Erro interno ao processar a requisição")
	}
}

// queryInt lê parâmetro numérico de query com default
func queryInt(r *http.Request, name string, fallback int) int {
	if raw := r.URL.Query().Get(name); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			return v
		}
	}
	return fallback
}
