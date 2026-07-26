package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/domain"
)

// colunas de metadados (listagens não trafegam content_md)
const metaColumns = `id, slug, title, description, lang, tags, cover_url, status, published_at, created_at, updated_at`

type PostgresArticleRepository struct {
	db *sqlx.DB
}

func NewPostgresArticleRepository(db *sqlx.DB) *PostgresArticleRepository {
	return &PostgresArticleRepository{db: db}
}

// articleRow espelha as colunas do banco (tags text[] exige pq.StringArray no scan)
type articleRow struct {
	ID          string         `db:"id"`
	Slug        string         `db:"slug"`
	Title       string         `db:"title"`
	Description sql.NullString `db:"description"`
	Lang        string         `db:"lang"`
	Tags        pq.StringArray `db:"tags"`
	CoverURL    sql.NullString `db:"cover_url"`
	ContentMD   sql.NullString `db:"content_md"`
	Status      string         `db:"status"`
	PublishedAt sql.NullTime   `db:"published_at"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

func (r articleRow) toDomain() *domain.Article {
	article := &domain.Article{
		ID:        r.ID,
		Slug:      r.Slug,
		Title:     r.Title,
		Lang:      r.Lang,
		Tags:      []string(r.Tags),
		ContentMD: r.ContentMD.String,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	if article.Tags == nil {
		article.Tags = []string{}
	}
	if r.Description.Valid {
		article.Description = &r.Description.String
	}
	if r.CoverURL.Valid {
		article.CoverURL = &r.CoverURL.String
	}
	if r.PublishedAt.Valid {
		publishedAt := r.PublishedAt.Time
		article.PublishedAt = &publishedAt
	}
	return article
}

func (r *PostgresArticleRepository) Create(ctx context.Context, article *domain.Article) error {
	query := `INSERT INTO articles (slug, title, description, lang, tags, cover_url, content_md, status)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id, published_at, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		article.Slug, article.Title, article.Description, article.Lang,
		pq.Array(article.Tags), article.CoverURL, article.ContentMD, article.Status,
	).Scan(&article.ID, &article.PublishedAt, &article.CreatedAt, &article.UpdatedAt)
}

func (r *PostgresArticleRepository) Update(ctx context.Context, article *domain.Article) error {
	query := `UPDATE articles
	          SET slug = $1, title = $2, description = $3, lang = $4, tags = $5,
	              cover_url = $6, content_md = $7, updated_at = now()
	          WHERE id = $8
	          RETURNING updated_at`
	return r.db.QueryRowxContext(ctx, query,
		article.Slug, article.Title, article.Description, article.Lang,
		pq.Array(article.Tags), article.CoverURL, article.ContentMD, article.ID,
	).Scan(&article.UpdatedAt)
}

func (r *PostgresArticleRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM articles WHERE id = $1`, id)
	return err
}

func (r *PostgresArticleRepository) FindByID(ctx context.Context, id string) (*domain.Article, error) {
	return r.findOne(ctx, `SELECT `+metaColumns+`, content_md FROM articles WHERE id = $1`, id)
}

func (r *PostgresArticleRepository) FindPublishedBySlug(ctx context.Context, slug string) (*domain.Article, error) {
	return r.findOne(ctx,
		`SELECT `+metaColumns+`, content_md FROM articles WHERE slug = $1 AND status = 'published'`, slug)
}

func (r *PostgresArticleRepository) findOne(ctx context.Context, query string, arg any) (*domain.Article, error) {
	var row articleRow
	if err := r.db.GetContext(ctx, &row, query, arg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *PostgresArticleRepository) List(ctx context.Context, filter domain.ListFilter) ([]*domain.Article, int, error) {
	where := ``
	if filter.OnlyPublished {
		where = `WHERE status = 'published'`
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT count(*) FROM articles `+where); err != nil {
		return nil, 0, err
	}

	// publicados por data de publicação; drafts (admin) entram pela data de criação
	query := `SELECT ` + metaColumns + ` FROM articles ` + where + `
	          ORDER BY COALESCE(published_at, created_at) DESC
	          LIMIT $1 OFFSET $2`
	var rows []articleRow
	if err := r.db.SelectContext(ctx, &rows, query, filter.Size, filter.Offset()); err != nil {
		return nil, 0, err
	}

	articles := make([]*domain.Article, 0, len(rows))
	for _, row := range rows {
		articles = append(articles, row.toDomain())
	}
	return articles, total, nil
}

func (r *PostgresArticleRepository) SlugExists(ctx context.Context, slug string, excludeID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM articles WHERE slug = $1 AND ($2 = '' OR id::text <> $2))`
	err := r.db.GetContext(ctx, &exists, query, slug, excludeID)
	return exists, err
}

func (r *PostgresArticleRepository) SetStatus(ctx context.Context, id string, status string) (*domain.Article, error) {
	// primeira publicação carimba published_at; republicar preserva a data original
	query := `UPDATE articles
	          SET status = $1,
	              published_at = CASE WHEN $1 = 'published' THEN COALESCE(published_at, now()) ELSE published_at END,
	              updated_at = now()
	          WHERE id = $2
	          RETURNING ` + metaColumns + `, content_md`
	var row articleRow
	if err := r.db.GetContext(ctx, &row, query, status, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row.toDomain(), nil
}
