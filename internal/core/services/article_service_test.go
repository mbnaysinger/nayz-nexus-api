package services

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Como publicar um artigo neste portfólio": "como-publicar-um-artigo-neste-portfolio",
		"IA aplicada, do desenho à produção.":     "ia-aplicada-do-desenho-a-producao",
		"  Espaços   e -- hífens ":                "espacos-e-hifens",
		"Ação & Reação: 100%":                     "acao-reacao-100",
		"":                                        "",
	}
	for input, want := range cases {
		if got := Slugify(input); got != want {
			t.Errorf("Slugify(%q) = %q; esperado %q", input, got, want)
		}
	}
}
