package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/domain"
)

var (
	ErrInvalidMedia       = errors.New("mídia inválida")
	ErrStorageUnavailable = errors.New("storage de mídia não configurado")
)

// Tipos aceitos (Q4 do spec: imagens + vídeo mp4/webm) — content-type → extensão canônica
var allowedMediaTypes = map[string]string{
	"image/png":     "png",
	"image/jpeg":    "jpg",
	"image/webp":    "webp",
	"image/gif":     "gif",
	"image/svg+xml": "svg",
	"video/mp4":     "mp4",
	"video/webm":    "webm",
}

type MediaService struct {
	storage domain.MediaStorage
}

func NewMediaService(storage domain.MediaStorage) *MediaService {
	return &MediaService{storage: storage}
}

// Presign valida tipo/nome e devolve URL de upload direto (PUT) + URL pública.
// O content-type é fixado na assinatura: o upload só passa com o tipo declarado.
func (s *MediaService) Presign(ctx context.Context, filename, contentType string) (*domain.PresignedUpload, error) {
	if s == nil || s.storage == nil {
		return nil, ErrStorageUnavailable
	}

	ext, ok := allowedMediaTypes[strings.ToLower(strings.TrimSpace(contentType))]
	if !ok {
		return nil, fmt.Errorf("%w: content_type %q não permitido", ErrInvalidMedia, contentType)
	}
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("%w: filename é obrigatório", ErrInvalidMedia)
	}

	base := Slugify(strings.TrimSuffix(path.Base(filename), path.Ext(filename)))
	if base == "" {
		base = "arquivo"
	}
	if len(base) > 40 {
		base = strings.Trim(base[:40], "-")
	}

	// chave particionada por ano/mês + sufixo aleatório (evita colisão e enumeração)
	key := fmt.Sprintf("articles/%s/%s-%s.%s", time.Now().UTC().Format("2006/01"), randomHex(8), base, ext)
	return s.storage.PresignUpload(ctx, key, contentType)
}

func randomHex(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
