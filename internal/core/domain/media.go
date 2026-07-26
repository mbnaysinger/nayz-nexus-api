package domain

import "context"

// PresignedUpload é o contrato devolvido ao admin: o browser sobe o arquivo
// direto para o MinIO via upload_url (PUT) e referencia public_url no markdown.
type PresignedUpload struct {
	UploadURL        string `json:"upload_url"`
	PublicURL        string `json:"public_url"`
	Key              string `json:"key"`
	ExpiresInSeconds int    `json:"expires_in"`
}

// MediaStorage é a porta para o storage de mídias (adapter: MinIO)
type MediaStorage interface {
	PresignUpload(ctx context.Context, key string, contentType string) (*PresignedUpload, error)
}
