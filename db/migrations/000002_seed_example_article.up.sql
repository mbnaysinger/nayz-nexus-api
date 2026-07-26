INSERT INTO articles (slug, title, description, lang, tags, content_md, status, published_at)
VALUES (
  'como-publicar-um-artigo',
  'Como publicar um artigo neste portfólio',
  'Artigo de exemplo — mostra o formato markdown suportado pela plataforma de conteúdo da nayztech. Substitua ou exclua quando publicar o primeiro texto real.',
  'pt',
  ARRAY['meta', 'markdown'],
  $md$Este é um artigo de exemplo. Os textos da nayztech são escritos em **markdown**, persistidos em Postgres e publicados pela área admin — a listagem em `/artigos` segue a ordem cronológica de publicação.

## O que o markdown suporta

Texto com **negrito**, *itálico* e [links](https://github.com/mbnaysinger). Listas:

- Arquitetura de soluções
- IA aplicada ao ciclo de desenvolvimento
- Multi-cloud e plataformas

Blocos de código com syntax highlight:

```java
@RestController
public class HealthController {
    @GetMapping("/health")
    public Map<String, String> health() {
        return Map.of("status", "UP");
    }
}
```

> Citações também funcionam — úteis para destacar decisões de arquitetura e trade-offs.

Imagens e vídeos entram por upload direto ao MinIO (o editor gera a URL pública automaticamente).

Bom texto!$md$,
  'published',
  '2026-07-25T12:00:00-03:00'
)
ON CONFLICT (slug) DO NOTHING;
