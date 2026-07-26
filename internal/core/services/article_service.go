package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mbnaysinger/nayz-nexus-api/internal/core/domain"
)

var (
	ErrInvalidArticle  = errors.New("artigo inválido")
	ErrArticleNotFound = errors.New("artigo não encontrado")
	ErrSlugConflict    = errors.New("slug já utilizado por outro artigo")
)

// ArticleInput é o payload de criação/edição vindo da área admin
type ArticleInput struct {
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	Lang        string   `json:"lang"`
	Tags        []string `json:"tags"`
	CoverURL    string   `json:"cover_url"`
	ContentMD   string   `json:"content_md"`
}

type ArticleService struct {
	articleRepo domain.ArticleRepository
}

func NewArticleService(articleRepo domain.ArticleRepository) *ArticleService {
	return &ArticleService{articleRepo: articleRepo}
}

// ---- público ----

func (s *ArticleService) ListPublished(ctx context.Context, page, size int) ([]*domain.Article, int, error) {
	filter := normalizeFilter(domain.ListFilter{OnlyPublished: true, Page: page, Size: size})
	return s.articleRepo.List(ctx, filter)
}

func (s *ArticleService) GetPublishedBySlug(ctx context.Context, slug string) (*domain.Article, error) {
	article, err := s.articleRepo.FindPublishedBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, ErrArticleNotFound
	}
	return article, nil
}

// ---- admin ----

func (s *ArticleService) ListAll(ctx context.Context, page, size int) ([]*domain.Article, int, error) {
	filter := normalizeFilter(domain.ListFilter{OnlyPublished: false, Page: page, Size: size})
	return s.articleRepo.List(ctx, filter)
}

func (s *ArticleService) GetByID(ctx context.Context, id string) (*domain.Article, error) {
	article, err := s.articleRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, ErrArticleNotFound
	}
	return article, nil
}

func (s *ArticleService) Create(ctx context.Context, input ArticleInput) (*domain.Article, error) {
	article, err := s.buildArticle(ctx, input, "")
	if err != nil {
		return nil, err
	}
	if err := s.articleRepo.Create(ctx, article); err != nil {
		return nil, err
	}
	return article, nil
}

func (s *ArticleService) Update(ctx context.Context, id string, input ArticleInput) (*domain.Article, error) {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	updated, err := s.buildArticle(ctx, input, existing.ID)
	if err != nil {
		return nil, err
	}
	existing.Title = updated.Title
	existing.Slug = updated.Slug
	existing.Description = updated.Description
	existing.Lang = updated.Lang
	existing.Tags = updated.Tags
	existing.CoverURL = updated.CoverURL
	existing.ContentMD = updated.ContentMD

	if err := s.articleRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *ArticleService) Delete(ctx context.Context, id string) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.articleRepo.Delete(ctx, id)
}

// Publish seta published_at na primeira publicação (republicar preserva a data
// original — a ordem cronológica da listagem não muda)
func (s *ArticleService) Publish(ctx context.Context, id string) (*domain.Article, error) {
	return s.setStatus(ctx, id, domain.StatusPublished)
}

func (s *ArticleService) Unpublish(ctx context.Context, id string) (*domain.Article, error) {
	return s.setStatus(ctx, id, domain.StatusDraft)
}

func (s *ArticleService) setStatus(ctx context.Context, id string, status string) (*domain.Article, error) {
	article, err := s.articleRepo.SetStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}
	if article == nil {
		return nil, ErrArticleNotFound
	}
	return article, nil
}

// ---- helpers ----

func (s *ArticleService) buildArticle(ctx context.Context, input ArticleInput, excludeID string) (*domain.Article, error) {
	title := strings.TrimSpace(input.Title)
	if len(title) < 3 {
		return nil, fmt.Errorf("%w: título é obrigatório (mínimo 3 caracteres)", ErrInvalidArticle)
	}
	if strings.TrimSpace(input.ContentMD) == "" {
		return nil, fmt.Errorf("%w: content_md é obrigatório", ErrInvalidArticle)
	}

	slug := Slugify(input.Slug)
	if slug == "" {
		slug = Slugify(title)
	}
	if slug == "" {
		return nil, fmt.Errorf("%w: não foi possível derivar um slug do título", ErrInvalidArticle)
	}
	taken, err := s.articleRepo.SlugExists(ctx, slug, excludeID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("%w: %s", ErrSlugConflict, slug)
	}

	lang := strings.TrimSpace(input.Lang)
	if lang == "" {
		lang = "pt"
	}

	tags := make([]string, 0, len(input.Tags))
	for _, tag := range input.Tags {
		if t := strings.TrimSpace(tag); t != "" {
			tags = append(tags, t)
		}
	}

	article := &domain.Article{
		Slug:      slug,
		Title:     title,
		Lang:      lang,
		Tags:      tags,
		ContentMD: input.ContentMD,
		Status:    domain.StatusDraft,
	}
	if desc := strings.TrimSpace(input.Description); desc != "" {
		article.Description = &desc
	}
	if cover := strings.TrimSpace(input.CoverURL); cover != "" {
		article.CoverURL = &cover
	}
	return article, nil
}

func normalizeFilter(f domain.ListFilter) domain.ListFilter {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 {
		f.Size = 10
	}
	if f.Size > 50 {
		f.Size = 50
	}
	return f
}

var slugReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// Slugify normaliza um texto para URL: minúsculas, sem acentos, hífens
func Slugify(s string) string {
	s = slugReplacer.Replace(strings.ToLower(strings.TrimSpace(s)))
	var b strings.Builder
	lastDash := true // evita hífen no início
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
