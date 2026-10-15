package indexing

import (
	"os"
	"os/exec"
	"testing"
)

// Testa a escolha do OCR no construtor de producao: NewDeterministicIndexer
// chamava so o tesseract local e THINGINDEX_OCR_ENDPOINT era ignorado sem erro.

func exigirFerramentasDeOCR(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm ausente: o construtor devolve nil por falta de renderizador, nao por escolha de provider")
	}
}

func TestIndexadorDeProducaoUsaOProviderRemotoQuandoConfigurado(t *testing.T) {
	exigirFerramentasDeOCR(t)

	t.Setenv("THINGINDEX_OCR_ENDPOINT", "http://ocr.invalido.local/v1/chat/completions")

	indexer := NewDeterministicIndexer()
	if indexer.pdfOCRExtractor == nil {
		t.Fatal("indexador sem extrator de OCR mesmo com endpoint configurado")
	}

	if nome := indexer.pdfOCRExtractor.Name(); nome != unlimitedOCREngineName {
		t.Fatalf("com THINGINDEX_OCR_ENDPOINT definido o indexador deveria usar %q, usou %q",
			unlimitedOCREngineName, nome)
	}
}

func TestIndexadorDeProducaoCaiParaOLocalSemEndpoint(t *testing.T) {
	exigirFerramentasDeOCR(t)

	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract ausente: sem ele o fallback local nao existe nesta maquina")
	}

	// Garante ausencia mesmo que o ambiente do desenvolvedor exporte a variavel.
	t.Setenv("THINGINDEX_OCR_ENDPOINT", "")

	indexer := NewDeterministicIndexer()
	if indexer.pdfOCRExtractor == nil {
		t.Fatal("indexador ficou sem OCR nenhum: o fallback local sumiu")
	}

	if nome := indexer.pdfOCRExtractor.Name(); nome != "local_tesseract" {
		t.Fatalf("sem endpoint o indexador deveria usar o tesseract local, usou %q", nome)
	}
}

// Variavel presente mas vazia (.env copiado do exemplo) nao ativa o remoto.
func TestEndpointEmBrancoNaoAtivaOProviderRemoto(t *testing.T) {
	exigirFerramentasDeOCR(t)

	for _, valor := range []string{"", "   ", "\t"} {
		t.Setenv("THINGINDEX_OCR_ENDPOINT", valor)

		if extrator := newUnlimitedOCRExtractor(); extrator != nil {
			t.Fatalf("endpoint %q ativou o provider remoto; deveria ser tratado como ausente", valor)
		}
	}
}

// Name() vai para ocr_engine no indice; os dois providers precisam diferir.
func TestNomeDoProviderRemotoNaoColideComOLocal(t *testing.T) {
	if unlimitedOCREngineName == "local_tesseract" {
		t.Fatal("provider remoto usa o mesmo nome do local: ocr_engine deixa de identificar o motor")
	}
	if os.Getenv("THINGINDEX_OCR_ENDPOINT") != "" {
		t.Log("aviso: ambiente tem THINGINDEX_OCR_ENDPOINT definido")
	}
}
