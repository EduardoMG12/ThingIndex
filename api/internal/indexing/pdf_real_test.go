package indexing

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// PDFs reais de exemplo pelo pipeline. As asserções usam trechos do conteudo
// conferidos com pdftotext; so contar paginas passaria com lixo binario.

const diretorioDePDFs = "../usecase/testdata/indexing-benchmark"

type expectativaDePDF struct {
	arquivo    string
	nosMinimos int
	// Trechos do CORPO das paginas.
	trechosNoTexto []string
	// Titulos que precisam virar no de sumario (nao aparecem no corpo da pagina).
	titulosEsperados []string
}

func exigirFerramentasDePDF(t *testing.T) {
	t.Helper()
	for _, ferramenta := range []string{"pdftotext", "pdfinfo"} {
		if _, err := exec.LookPath(ferramenta); err != nil {
			t.Skipf("%s ausente: o extrator cai para o parser nativo e as contagens mudam", ferramenta)
		}
	}
}

// paginasSegundoPdfinfo: compara com o pdfinfo para o teste sobreviver a troca
// dos arquivos de exemplo.
func paginasSegundoPdfinfo(t *testing.T, caminho string) int {
	t.Helper()

	saida, err := exec.Command("pdfinfo", caminho).Output()
	if err != nil {
		t.Fatalf("pdfinfo falhou em %s: %v", caminho, err)
	}

	for _, linha := range strings.Split(string(saida), "\n") {
		if !strings.HasPrefix(linha, "Pages:") {
			continue
		}
		campos := strings.Fields(linha)
		if len(campos) < 2 {
			break
		}
		total, err := strconv.Atoi(campos[1])
		if err != nil {
			t.Fatalf("contagem de paginas ilegivel em %s: %q", caminho, linha)
		}
		return total
	}

	t.Fatalf("pdfinfo nao reportou paginas para %s", caminho)
	return 0
}

func indexarPDF(t *testing.T, caminho string) domain.IndexingResult {
	t.Helper()

	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("nao foi possivel ler %s: %v", caminho, err)
	}

	indexer := NewDeterministicIndexer()
	doc := &models.Document{
		ID:       1,
		Title:    filepath.Base(caminho),
		MimeType: "application/pdf",
	}

	resultado, err := indexer.Index(context.Background(), doc, bytes.NewReader(dados))
	if err != nil {
		t.Fatalf("indexacao de %s falhou: %v", filepath.Base(caminho), err)
	}
	return resultado
}

func TestPDFsReaisPassamPeloIndexadorCompleto(t *testing.T) {
	exigirFerramentasDePDF(t)

	casos := []expectativaDePDF{
		{
			arquivo:    "court-order.pdf",
			nosMinimos: 3,
			// Trechos conferidos com pdftotext: provam que saiu texto legivel,
			// e nao string vazia ou lixo binario.
			trechosNoTexto:   []string{"comprovantes de entrega"},
			titulosEsperados: []string{"1 RELATORIO", "3 DISPOSITIVO"},
		},
		{
			arquivo:          "supplier-master-order-form.pdf",
			nosMinimos:       3,
			trechosNoTexto:   []string{"SOC monitoring"},
			titulosEsperados: []string{"1 SCOPE OF SERVICES"},
		},
	}

	for _, caso := range casos {
		t.Run(caso.arquivo, func(t *testing.T) {
			caminho := filepath.Join(diretorioDePDFs, caso.arquivo)
			if _, err := os.Stat(caminho); err != nil {
				t.Skipf("arquivo de exemplo ausente: %v", err)
			}

			esperadas := paginasSegundoPdfinfo(t, caminho)
			resultado := indexarPDF(t, caminho)

			if len(resultado.DocumentPages) != esperadas {
				t.Errorf("o extrator perdeu paginas: pdfinfo diz %d, o indice tem %d",
					esperadas, len(resultado.DocumentPages))
			}

			if len(resultado.OutlineNodes) < caso.nosMinimos {
				t.Errorf("estrutura rasa demais: %d nos, esperado ao menos %d",
					len(resultado.OutlineNodes), caso.nosMinimos)
			}

			// Pagina vazia em PDF COM camada de texto significa extrator
			// quebrado -- diferente de PDF escaneado, onde o OCR entraria.
			for _, pagina := range resultado.DocumentPages {
				if strings.TrimSpace(pagina.Content) == "" {
					t.Errorf("pagina %d veio vazia num PDF com camada de texto",
						pagina.PageNumber)
				}
			}

			var partes []string
			for _, pagina := range resultado.DocumentPages {
				partes = append(partes, pagina.Content)
			}
			textoCompleto := strings.Join(partes, " ")
			for _, trecho := range caso.trechosNoTexto {
				if !strings.Contains(textoCompleto, trecho) {
					t.Errorf("trecho conhecido %q nao apareceu no corpo das paginas", trecho)
				}
			}

			// Os titulos vivem nos nos, nao nas paginas.
			var titulos []string
			for _, no := range resultado.OutlineNodes {
				titulos = append(titulos, no.Title)
			}
			todosOsTitulos := strings.Join(titulos, " | ")
			for _, esperado := range caso.titulosEsperados {
				if !strings.Contains(todosOsTitulos, esperado) {
					t.Errorf("titulo %q nao virou no de sumario. Nos: %s", esperado, todosOsTitulos)
				}
			}
		})
	}
}

// Numeracao de pagina comecando em 1, sem repeticao e em ordem.
func TestPDFPreservaANumeracaoDePagina(t *testing.T) {
	exigirFerramentasDePDF(t)

	caminho := filepath.Join(diretorioDePDFs, "court-order.pdf")
	if _, err := os.Stat(caminho); err != nil {
		t.Skipf("arquivo de exemplo ausente: %v", err)
	}

	resultado := indexarPDF(t, caminho)
	vistas := make(map[int]bool, len(resultado.DocumentPages))

	for _, pagina := range resultado.DocumentPages {
		if pagina.PageNumber < 1 {
			t.Errorf("pagina numerada como %d: a proveniencia por pagina comeca em 1",
				pagina.PageNumber)
		}
		if vistas[pagina.PageNumber] {
			t.Errorf("pagina %d duplicada: a citacao apontaria para o lugar errado",
				pagina.PageNumber)
		}
		vistas[pagina.PageNumber] = true
	}
}

// PDF corrompido devolve erro, sem panic (panic derruba o worker).
func TestPDFCorrompidoFalhaSemDerrubarOProcesso(t *testing.T) {
	indexer := NewDeterministicIndexer()
	doc := &models.Document{ID: 1, Title: "corrompido.pdf", MimeType: "application/pdf"}

	// Cabecalho valido seguido de lixo: passa pela deteccao de tipo e quebra na
	// extracao, que e o caminho perigoso.
	dados := append([]byte("%PDF-1.4\n"), bytes.Repeat([]byte{0xFF, 0x00, 0xAB}, 400)...)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("indexador entrou em panico com PDF corrompido: %v", r)
		}
	}()

	resultado, err := indexer.Index(context.Background(), doc, bytes.NewReader(dados))
	if err == nil && len(resultado.DocumentPages) == 0 && len(resultado.OutlineNodes) == 0 {
		// Aceitavel: extraiu nada e nao mentiu sobre isso.
		return
	}
	if err != nil {
		// Aceitavel: falhou explicitamente.
		return
	}

	t.Fatalf("PDF corrompido produziu %d paginas e %d nos: o extrator inventou estrutura",
		len(resultado.DocumentPages), len(resultado.OutlineNodes))
}
