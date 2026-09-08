package indexing

import (
	"os"
	"os/exec"
	"testing"
)

// O provider de OCR remoto (THINGINDEX_OCR_ENDPOINT) existia, estava testado e
// NUNCA era alcancado em producao: NewDeterministicIndexer chamava
// newCommandPDFOCRExtractor() -- so o tesseract local -- em vez de
// newPDFOCRExtractor(), que tenta o remoto e cai para o local.
//
// O efeito era silencioso e caro de perceber: configurar o endpoint nao mudava
// nada, o indice sempre gravava ocr_engine='local_tesseract', e o servico `ocr`
// do docker-compose ficava inalcancavel. Nenhum erro, nenhum log -- so a
// funcionalidade ausente.
//
// Estes testes travam a decisao no construtor de PRODUCAO, que era onde o
// defeito morava. Testar newPDFOCRExtractor isoladamente nao teria pegado nada:
// aquela funcao sempre esteve correta.

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

// TestEndpointEmBrancoNaoAtivaOProviderRemoto cobre o caso de configuracao
// pela metade: variavel presente mas vazia, comum em .env copiado de exemplo.
// Ativar o remoto com endpoint vazio faria toda indexacao de PDF escaneado
// falhar por erro de rede.
func TestEndpointEmBrancoNaoAtivaOProviderRemoto(t *testing.T) {
	exigirFerramentasDeOCR(t)

	for _, valor := range []string{"", "   ", "\t"} {
		t.Setenv("THINGINDEX_OCR_ENDPOINT", valor)

		if extrator := newUnlimitedOCRExtractor(); extrator != nil {
			t.Fatalf("endpoint %q ativou o provider remoto; deveria ser tratado como ausente", valor)
		}
	}
}

// TestNomeDoProviderRemotoNaoColideComOLocal protege a metrica: o valor de
// Name() e gravado como ocr_engine no indice e e a unica evidencia de qual
// motor processou o documento. Dois providers com o mesmo nome tornariam a
// auditoria incapaz de distinguir.
func TestNomeDoProviderRemotoNaoColideComOLocal(t *testing.T) {
	if unlimitedOCREngineName == "local_tesseract" {
		t.Fatal("provider remoto usa o mesmo nome do local: ocr_engine deixa de identificar o motor")
	}
	if os.Getenv("THINGINDEX_OCR_ENDPOINT") != "" {
		t.Log("aviso: ambiente tem THINGINDEX_OCR_ENDPOINT definido")
	}
}
