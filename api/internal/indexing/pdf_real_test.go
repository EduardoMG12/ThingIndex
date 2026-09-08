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

// Quase toda a caçada de defeitos desta rodada usou HTML, que e o caminho facil
// do extrator. PDF e o formato que o trabalho promete tratar e o unico que
// exercita pdftotext, deteccao de pagina e OCR.
//
// Enviei estes arquivos pelo pipeline manualmente antes de escrever o teste:
// 2 paginas cada, 4 nos cada, zero paginas vazias. O pipeline FUNCIONA -- e e
// justamente por isso que vale travar, porque a proxima regressao no extrator
// passaria despercebida.
//
// As asserções usam trechos REAIS do conteudo, verificados com pdftotext. Um
// teste que so conta paginas passaria com o extrator devolvendo lixo binario.

const diretorioDePDFs = "../usecase/testdata/indexing-benchmark"

type expectativaDePDF struct {
	arquivo    string
	paginas    int
	nosMinimos int
	// Trechos do CORPO das paginas.
	trechosNoTexto []string
	// Titulos que precisam virar no de sumario. O extrator separa estrutura de
	// conteudo: cabecalho vira no, o resto vira pagina -- e afirmar o titulo no
	// corpo da pagina foi erro meu na primeira versao deste teste.
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

// paginasSegundoPdfinfo le a contagem da ferramenta externa.
//
// Comparar com o pdfinfo, em vez de um numero fixo no teste, faz a asserção
// continuar valendo se alguem trocar os arquivos de exemplo -- e detecta o
// extrator perdendo paginas, que era o objetivo.
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

			// Os titulos vivem nos nos, nao nas paginas: perder essa separacao
			// significa que a recuperacao em nivel de secao deixou de existir.
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

// TestPDFPreservaANumeracaoDePagina protege a proveniencia: o trabalho afirma
// que todo item recuperado aponta para a pagina de origem, e a metrica TPC do
// artigo depende disso. Numeracao comecando em zero, repetida ou fora de ordem
// quebraria a afirmacao sem quebrar nenhum outro teste.
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

// TestPDFCorrompidoFalhaSemDerrubarOProcesso: bytes invalidos vem de upload de
// usuario. O indexador precisa devolver erro, nunca entrar em panico -- panico
// no worker derruba a fila inteira.
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
