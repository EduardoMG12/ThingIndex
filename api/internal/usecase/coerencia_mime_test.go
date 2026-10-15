package usecase

import (
	"testing"

	"thing-index/internal/indexing"
)

// O upload aceita PDF, HTML e XLSX; qualquer outro tipo cai no fallback de
// texto puro do indexador. Formato novo aceito no upload sem extrator
// dedicado seria indexado sem estrutura e sem erro.

// tiposComExtracaoDedicada sao os mime types que o indexador trata
// explicitamente, cada um com extrator proprio.
var tiposComExtracaoDedicada = []string{
	"application/pdf",
	"text/html",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

func TestTodoTipoAceitoNoUploadTemExtratorDedicado(t *testing.T) {
	dedicados := make(map[string]struct{}, len(tiposComExtracaoDedicada))
	for _, tipo := range tiposComExtracaoDedicada {
		dedicados[tipo] = struct{}{}
	}

	for tipo := range allowedUploadMimeTypes {
		if _, temExtrator := dedicados[tipo]; !temExtrator {
			t.Errorf("o upload aceita %q mas o indexador nao tem extrator dedicado.\n\n"+
				"O documento cairia no fallback de texto puro: seria indexado com "+
				"sucesso, porem sem sumario, sem tabela e sem pagina. O usuario veria "+
				"um documento vazio sem nenhuma mensagem de erro.\n\n"+
				"Ao adicionar um formato, trate-o no switch de DeterministicIndexer.Index "+
				"e acrescente aqui.", tipo)
		}
	}
}

func TestTodoExtratorDedicadoEhAceitoNoUpload(t *testing.T) {
	// Extrator que nunca recebe documento e codigo morto.
	for _, tipo := range tiposComExtracaoDedicada {
		if _, aceito := allowedUploadMimeTypes[tipo]; !aceito {
			t.Errorf("existe extrator dedicado para %q, mas o upload rejeita esse tipo: "+
				"o extrator e codigo morto", tipo)
		}
	}
}

func TestFallbackDeTextoPuroContinuaSeguro(t *testing.T) {
	indexer := indexing.NewDeterministicIndexer()
	if indexer == nil {
		t.Fatal("indexador nao foi construido")
	}

	// O fallback fica; a verificacao de comportamento vive em internal/indexing.
	if len(allowedUploadMimeTypes) == 0 {
		t.Fatal("a lista de tipos aceitos ficou vazia: nenhum upload seria possivel")
	}
}
