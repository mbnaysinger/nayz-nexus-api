# nayz-nexus-api

API de conteúdo do hub **nayztech** (nexus.nayztech.com.br): artigos em markdown persistidos em **Postgres**, mídias em **MinIO** via presigned URLs, área admin autenticada pelo **nayz-auth** (JWT HS256).

Arquitetura hexagonal, no mesmo modelo do `nayz-auth` e do `tallo-planning-api`:

```
cmd/nayz-nexus-api/       bootstrap (env, migrations, wiring)
internal/config/          conexão Postgres (sqlx + pgx)
internal/core/domain/     entidades e portas (ArticleRepository, MediaStorage)
internal/core/services/   regras de negócio (artigos, slug, presign)
internal/adapters/
  handlers/               HTTP (net/http + ServeMux com method routing)
  middlewares/            auth (JWT nayz-auth), CORS, logger, rate limit
  repositories/           Postgres (schema nayztech)
  storage/                MinIO (presigned PUT)
db/migrations/            versionadas, aplicadas automaticamente no start
```

## Rodando local

Postgres e MinIO **não sobem por compose** — usa as instâncias já existentes (local/VPS).

```bash
cp .env.example .env   # preencha DATABASE_URL, JWT_SECRET etc.
go run ./cmd/nayz-nexus-api
```

No start a API aplica as migrations (cria o schema `nayztech`, a tabela `articles` e um artigo de exemplo publicado). Sem credenciais de MinIO o serviço sobe normalmente — apenas o presign de mídia responde 503.

## Endpoints

**Público (sem auth, rate limit por IP):**

| Rota | Descrição |
|---|---|
| `GET /api/v1/articles?page=1&size=10` | publicados, `published_at DESC`, sem `content_md` |
| `GET /api/v1/articles/{slug}` | artigo publicado completo |
| `GET /health` | health check |

**Admin (Bearer JWT do nayz-auth — app NAYZTECH, role `ADMIN`):**

| Rota | Descrição |
|---|---|
| `GET /api/v1/admin/articles` | todos, inclusive drafts |
| `GET /api/v1/admin/articles/{id}` | completo (editor) |
| `POST /api/v1/admin/articles` | cria (nasce draft) |
| `PUT /api/v1/admin/articles/{id}` | atualiza |
| `DELETE /api/v1/admin/articles/{id}` | exclui |
| `PATCH /api/v1/admin/articles/{id}/publish` | publica (1ª publicação carimba `published_at`) |
| `PATCH /api/v1/admin/articles/{id}/unpublish` | volta a draft |
| `POST /api/v1/admin/media/presign` | `{filename, content_type}` → `{upload_url, public_url, key, expires_in}` |

Upload de mídia: o browser faz `PUT upload_url` com o arquivo (mesmo `Content-Type` do presign — o tipo é fixado na assinatura) e referencia `public_url` no markdown. Tipos aceitos: png, jpg, webp, gif, svg, mp4, webm.

```bash
# exemplo: criar e publicar um artigo
TOKEN=... # login na app NAYZTECH via nayz-auth
curl -X POST localhost:8082/api/v1/admin/articles \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"title":"Meu artigo","description":"resumo","tags":["arquitetura"],"content_md":"# Olá"}'
curl -X PATCH localhost:8082/api/v1/admin/articles/<id>/publish -H "Authorization: Bearer $TOKEN"
```

## Segurança (spec §9)

- JWT validado no backend: assinatura HS256 (`JWT_SECRET` idêntica à do nayz-auth) + `exp` + **`app_id`** (`NAYZ_AUTH_APP_ID`) + role `ADMIN`
- CORS restrito às origens de `CORS_ORIGINS`
- Rate limit por IP nas rotas públicas (`RATE_LIMIT_RPS`/`RATE_LIMIT_BURST`)
- Presigned URLs com expiração curta (`PRESIGN_EXPIRY_MINUTES`, default 10) e content-type fixado
- Mídia pública: no start, `EnsureBucket` aplica leitura anônima (somente `s3:GetObject`) no prefixo `articles/*` do bucket (`MINIO_PUBLIC_READ`, default true) — URLs estáveis com cache/SEO; upload/list/delete seguem autenticados. Atenção: a política do bucket é substituída por essa regra.

## Registro da aplicação NAYZTECH no nayz-auth

Adicionar como próxima migration de seed no repo `nayz-auth` (mesmo modelo da `000005_seed_tallo_planning_app`):

```sql
-- Cria a aplicação NAYZTECH com a role ADMIN (gerencia artigos do nexus).
DO $$
DECLARE
    v_app_id UUID;
BEGIN
    INSERT INTO applications (name, auth_methods, require_person)
    VALUES ('NAYZTECH', '{"PASSWORD"}', TRUE)
    RETURNING id INTO v_app_id;

    INSERT INTO roles (application_id, name) VALUES (v_app_id, 'ADMIN');
END $$;
```

O `id` gerado é o `NAYZ_AUTH_APP_ID` desta API e o `VITE_NAYZ_AUTH_APP_ID` do front. Vincule seu usuário à aplicação (`user_applications`) e à role ADMIN.

## Deploy (VPS)

Ver `../deploy/docker-compose.yml`: a API roda em container, alcança o Postgres/MinIO existentes via `host.docker.internal` (ou rede interna docker) e fica atrás do reverse proxy. A rota pública de mídia (`MINIO_PUBLIC_URL`) é servida pelo proxy apontando para o MinIO.
