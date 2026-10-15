package dto

import (
	"encoding/json"
	"testing"

	"thing-index/internal/domain"
)

func TestRelatorioNuloDevolveNulo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com relatorio nulo: %v", r)
		}
	}()

	if ToIndexingBenchmarkReportResponse(nil) != nil {
		t.Fatal("deveria devolver nil")
	}
}

func TestMetricasDoBenchmarkSobrevivemSemArredondar(t *testing.T) {
	// As metricas nao podem ser arredondadas na conversao.
	resposta := ToIndexingBenchmarkReportResponse(&domain.IndexingBenchmarkReport{
		DatasetVersion: "2026-03-20",
		Metrics: domain.IndexingBenchmarkMetrics{
			CorpusDocuments:      12,
			DocumentQueries:      48,
			StructurePassRate:    0.9166666666666666,
			TreeNodeTop1Accuracy: 0.5625,
			TreeNodeRecallAt3:    0.4583333333333333,
		},
	})

	if resposta == nil {
		t.Fatal("conversao devolveu nil")
	}
	if resposta.DatasetVersion != "2026-03-20" {
		t.Fatalf("versao do conjunto perdida: %q", resposta.DatasetVersion)
	}
	if resposta.Metrics.CorpusDocuments != 12 {
		t.Fatalf("contagem de documentos perdida: %d", resposta.Metrics.CorpusDocuments)
	}
	if resposta.Metrics.TreeNodeTop1Accuracy != 0.5625 {
		t.Fatalf("precisao arredondada na conversao: %v (esperado 0.5625)",
			resposta.Metrics.TreeNodeTop1Accuracy)
	}
	if resposta.Metrics.StructurePassRate != 0.9166666666666666 {
		t.Fatalf("taxa de estrutura arredondada: %v", resposta.Metrics.StructurePassRate)
	}
}

func TestListasDoRelatorioVaziasNaoViramNulas(t *testing.T) {
	resposta := ToIndexingBenchmarkReportResponse(&domain.IndexingBenchmarkReport{
		DatasetVersion: "2026-03-20",
	})

	bruto, err := json.Marshal(resposta)
	if err != nil {
		t.Fatalf("serializacao falhou: %v", err)
	}

	var mapa map[string]any
	if err := json.Unmarshal(bruto, &mapa); err != nil {
		t.Fatalf("JSON invalido: %v", err)
	}

	for _, chave := range []string{"corpus", "methodology", "pass_criteria", "limitations"} {
		valor, existe := mapa[chave]
		if !existe {
			continue
		}
		if valor == nil {
			t.Errorf("campo %q veio null: a pagina de benchmark quebra ao renderizar", chave)
		}
	}
}

func TestCorpusPreservaTitulosEsperados(t *testing.T) {
	// ExpectedOutlineTitles e o gabarito; perdido, o benchmark compara contra lista vazia.
	resposta := ToIndexingBenchmarkReportResponse(&domain.IndexingBenchmarkReport{
		Corpus: []domain.IndexingBenchmarkCorpusDocument{
			{
				ID:                    "court-order",
				Title:                 "Decisao judicial",
				MimeType:              "application/pdf",
				ExpectedOutlineTitles: []string{"1 RELATORIO", "2 FUNDAMENTACAO", "3 DISPOSITIVO"},
				OutlineNodeCount:      4,
			},
		},
	})

	if len(resposta.Corpus) != 1 {
		t.Fatalf("esperava 1 documento no corpus, veio %d", len(resposta.Corpus))
	}

	documento := resposta.Corpus[0]
	if len(documento.ExpectedOutlineTitles) != 3 {
		t.Fatalf("gabarito perdido: %v -- o benchmark compararia contra lista vazia e passaria sempre",
			documento.ExpectedOutlineTitles)
	}
	if documento.ExpectedOutlineTitles[0] != "1 RELATORIO" {
		t.Fatalf("gabarito corrompido: %v", documento.ExpectedOutlineTitles)
	}
	if documento.OutlineNodeCount != 4 {
		t.Fatalf("contagem de nos perdida: %d", documento.OutlineNodeCount)
	}
}

func TestCorpusCopiaOsTitulosEmVezDeCompartilharOSlice(t *testing.T) {
	// Sem a copia, alterar a resposta corromperia a fixture em memoria.
	original := []string{"1 RELATORIO", "2 FUNDAMENTACAO"}

	resposta := ToIndexingBenchmarkReportResponse(&domain.IndexingBenchmarkReport{
		Corpus: []domain.IndexingBenchmarkCorpusDocument{
			{ID: "doc", ExpectedOutlineTitles: original},
		},
	})

	resposta.Corpus[0].ExpectedOutlineTitles[0] = "ALTERADO"

	if original[0] != "1 RELATORIO" {
		t.Fatalf("a resposta compartilha memoria com a fixture: original virou %q", original[0])
	}
}
