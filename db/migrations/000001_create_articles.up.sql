CREATE TABLE articles (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  slug          text NOT NULL UNIQUE,          -- gerado do título, editável
  title         text NOT NULL,
  description   text,                          -- resumo para listagem/SEO
  lang          text NOT NULL DEFAULT 'pt',    -- artigos somente pt-BR (Q5)
  tags          text[] NOT NULL DEFAULT '{}',
  cover_url     text,                          -- capa (MinIO), opcional
  content_md    text NOT NULL,                 -- markdown (carro-chefe)
  status        text NOT NULL DEFAULT 'draft'  CHECK (status IN ('draft','published')),
  published_at  timestamptz,                   -- define a ordem cronológica
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_articles_published ON articles (status, published_at DESC);
