package usecase

import (
	"testing"

	"thing-index/internal/models"
)

// Linha curta com um so termo da pergunta nao vira fonte; a da matriz, com
// varios, continua vindo.
func TestLinhaComUmTermoSoNaoEFonte(t *testing.T) {
	t.Parallel()
	page := 1
	tables := matrizDeTeste()
	tables = append(tables, models.TableSection{ID: 10, DocumentID: 261, PageNumber: &page, Title: "Fevereiro 2026",
		Headers: []string{"Dia(s)", "Descrição"}, Rows: [][]string{{"2 a 6", "Equivalência de Estágio"}}})
	result := buildCollectionRetrieval("Qual é o pré-requisito de Estrutura de Dados I?",
		[]models.Document{{ID: 265, Title: "Matriz"}, {ID: 261, Title: "Calendario"}}, nil, nil, nil, tables)
	for _, hit := range result.TableRowHits {
		if hit.DocumentID == 261 {
			t.Fatalf("linha do calendario sem termo em comum virou fonte: %+v", hit)
		}
	}
	if len(result.TableRowHits) == 0 || result.TableRowHits[0].DocumentID != 265 {
		t.Fatalf("a linha da matriz deveria continuar sendo fonte: %+v", result.TableRowHits)
	}
}

func TestTermosDaPerguntaSaoNormalizadosAntesDeContar(t *testing.T) {
	t.Parallel()
	terms := significantQueryTokens("Quais são os pré-requisitos de Sistemas Operacionais?")
	if queryTermCoverage("Disciplina: SIS07305 - Sistemas Operacionais; Pré-requisito: SIS07201", terms) < 2 {
		t.Fatalf("'operacionais?' e 'Operacionais;' precisam casar: termos=%v", terms)
	}
}
