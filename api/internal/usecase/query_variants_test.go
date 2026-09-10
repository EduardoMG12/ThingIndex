package usecase

import (
	"reflect"
	"testing"

	"thing-index/internal/models"
)

func TestConsultaGanhaVarianteDeNumero(t *testing.T) {
	t.Parallel()
	got := expandNumberVariants([]string{"estagio", "extracurriculares", "resolucoes", "anuais", "curso", "sis07305", "de", "pre-requisitos"})
	want := []string{"estagio", "estagios", "extracurriculares", "extracurricular", "resolucoes", "resolucao", "anuais", "anual", "curso", "sis07305", "de", "pre-requisitos", "pre-requisito"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("variantes: %v", got)
	}
}

func TestLetraMaiusculaSoltaEhTermoDaConsulta(t *testing.T) {
	t.Parallel()
	tokens := tokenizeRetrievalText("O que significa o conceito D? A matrícula em TCC I e TCC II")
	tem := map[string]bool{}
	for _, tk := range tokens {
		tem[tk] = true
	}
	if !tem["d"] || !tem["i"] {
		t.Fatalf("letras maiusculas soltas deveriam entrar: %v", tokens)
	}
	if tem["a"] || tem["o"] {
		t.Fatalf("artigo minusculo continua fora: %v", tokens)
	}
}

func TestPluralNaPaginaCasaSingularNaPergunta(t *testing.T) {
	t.Parallel()
	pages := []models.DocumentPage{
		{DocumentID: 1, PageNumber: 199, Content: "para os estagios extracurriculares a carga horaria limite e de 80 horas por aula"},
		{DocumentID: 1, PageNumber: 5, Content: "o coordenador do curso e o colegiado decidem sobre a matricula e o estagio"},
	}
	result := buildCollectionRetrieval("Qual o limite de horas de estágio extracurricular?", []models.Document{{ID: 1, Title: "PPC"}}, pages, nil, nil, nil)
	if result == nil || len(result.PageHits) == 0 || result.PageHits[0].PageNumber != 199 {
		t.Fatalf("a pagina com o plural deveria vencer: %+v", result)
	}
}

func TestConceitoDAchaAPaginaCerta(t *testing.T) {
	t.Parallel()
	pages := []models.DocumentPage{
		{DocumentID: 1, PageNumber: 125, Content: "conceito A quando a aprendizagem for plena; conceito B quando parcialmente plena"},
		{DocumentID: 1, PageNumber: 126, Content: "conceito C quando suficiente; conceito D quando a aprendizagem for insuficiente"},
	}
	result := buildCollectionRetrieval("O que significa o conceito D?", []models.Document{{ID: 1, Title: "PPC"}}, pages, nil, nil, nil)
	if result == nil || len(result.PageHits) == 0 || result.PageHits[0].PageNumber != 126 {
		t.Fatalf("a pagina do conceito D deveria vencer: %+v", result)
	}
}
