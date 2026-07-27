package storage

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/domain"
)

// MinioStorage implementa domain.MediaStorage com presigned PUT URLs.
// O backend nunca trafega o arquivo: o browser sobe direto para o MinIO.
type MinioStorage struct {
	client     *minio.Client
	bucket     string
	publicBase string
	expiry     time.Duration
	publicRead bool
}

// NewMinioStorage conecta na instância existente (local ou VPS — nada é provisionado aqui).
// publicBase é a URL pública que serve a raiz do bucket (proxy reverso ou o próprio MinIO).
// publicRead aplica leitura anônima (somente GET) no prefixo articles/* a cada start.
func NewMinioStorage(endpoint, accessKey, secretKey, bucket, publicBase string, useSSL bool, expiry time.Duration, publicRead bool) (*MinioStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("cliente MinIO: %w", err)
	}
	return &MinioStorage{
		client:     client,
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
		expiry:     expiry,
		publicRead: publicRead,
	}, nil
}

// EnsureBucket cria o bucket caso não exista e garante a política de leitura
// anônima do prefixo articles/* (idempotente — roda a cada start).
func (s *MinioStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("verificando bucket %s: %w", s.bucket, err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("criando bucket %s: %w", s.bucket, err)
		}
		slog.Info("Bucket MinIO criado", slog.String("bucket", s.bucket))
	}
	if s.publicRead {
		return s.ensurePublicReadPolicy(ctx)
	}
	return nil
}

// ensurePublicReadPolicy libera somente s3:GetObject anônimo em articles/* —
// as URLs públicas dos artigos funcionam com cache/SEO e sem expiração, enquanto
// upload/list/delete seguem exigindo credencial (o presigned PUT não muda).
// Atenção: SetBucketPolicy SUBSTITUI a política do bucket por esta.
func (s *MinioStorage) ensurePublicReadPolicy(ctx context.Context) error {
	policy := fmt.Sprintf(`{
	  "Version": "2012-10-17",
	  "Statement": [{
	    "Effect": "Allow",
	    "Principal": {"AWS": ["*"]},
	    "Action": ["s3:GetObject"],
	    "Resource": ["arn:aws:s3:::%s/articles/*"]
	  }]
	}`, s.bucket)
	if err := s.client.SetBucketPolicy(ctx, s.bucket, policy); err != nil {
		return fmt.Errorf("aplicando política de leitura anônima em %s/articles/*: %w", s.bucket, err)
	}
	slog.Info("Política de leitura anônima garantida", slog.String("bucket", s.bucket), slog.String("prefixo", "articles/*"))
	return nil
}

func (s *MinioStorage) PresignUpload(ctx context.Context, key string, contentType string) (*domain.PresignedUpload, error) {
	// Content-Type entra na assinatura: o upload só é aceito com o tipo declarado
	headers := http.Header{"Content-Type": []string{contentType}}
	uploadURL, err := s.client.PresignHeader(ctx, http.MethodPut, s.bucket, key, s.expiry, url.Values{}, headers)
	if err != nil {
		return nil, fmt.Errorf("presign PUT %s: %w", key, err)
	}
	return &domain.PresignedUpload{
		UploadURL:        uploadURL.String(),
		PublicURL:        s.publicBase + "/" + key,
		Key:              key,
		ExpiresInSeconds: int(s.expiry.Seconds()),
	}, nil
}
