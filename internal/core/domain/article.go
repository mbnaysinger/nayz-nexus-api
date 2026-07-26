package domain

import (
	"context"
	"time"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

type Article struct {
	ID          string     `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Lang        string     `json:"lang"`
	Tags        []string   `json:"tags"`
	CoverURL    *string    `json:"cover_url"`
	ContentMD   string     `json:"content_md,omitempty"`
	Status      string     `json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListFilter parametriza listagens paginadas de artigos
type ListFilter struct {
	OnlyPublished bool
	Page          int
	Size          int
}

func (f ListFilter) Offset() int {
	return (f.Page - 1) * f.Size
}

type ArticleRepository interface {
	Create(ctx context.Context, article *Article) error
	Update(ctx context.Context, article *Article) error
	Delete(ctx context.Context, id string) error
	// FindByID retorna o artigo completo (com content_md); nil quando não existe
	FindByID(ctx context.Context, id string) (*Article, error)
	// FindPublishedBySlug retorna o artigo publicado completo; nil quando não existe
	FindPublishedBySlug(ctx context.Context, slug string) (*Article, error)
	// List retorna metadados (sem content_md) e o total para paginação
	List(ctx context.Context, filter ListFilter) ([]*Article, int, error)
	// SlugExists confere unicidade do slug ignorando o próprio artigo (excludeID vazio em criação)
	SlugExists(ctx context.Context, slug string, excludeID string) (bool, error)
	// SetStatus publica/despublica; retorna o artigo atualizado ou nil quando não existe
	SetStatus(ctx context.Context, id string, status string) (*Article, error)
}
