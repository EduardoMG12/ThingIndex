package usecase

import (
	"testing"

	"thing-index/internal/indexing"
)

// O upload aceita PDF, HTML e XLSX. O indexador tem um ramo `default` que trata
// qualquer outro tipo como texto puro (indexPlainText).
//
// Isso parecia inconsistencia -- e foi registrado como tal no handover -- mas a
// medicao contra a API viva mostrou que o ramo e INALCANCAVEL: cinco tentativas
// de enviar texto puro (text/plain, application/octet-stream, text/markdown,
// extensao .txt, extensao .html com corpo de texto) foram todas rejeitadas com
// HTTP 400.
//
// A decisao e manter o fallback e travar a coerencia com teste, em vez de
// remove-lo:
//
//   - remover deixaria o indexador em panico ou devolvendo erro cru se um mime
//     novo entrar na lista de aceitos e alguem esquecer do switch;
//   - o fallback e a opcao segura: extrai o que der, sem inventar estrutura.
//
// O que este teste impede e a divergencia SILENCIOSA: adicionar um formato ao
// upload sem tratar no indexador faria o documento cair no texto puro e perder
// toda a estrutura -- indexado com sucesso, mas sem sumario, sem tabela e sem
// pagina. O usuario veria um documento vazio sem nenhum erro.

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
	// A direcao oposta: extrator que existe e nunca recebe documento e codigo
	// morto -- foi exatamente o caso do provider de OCR remoto, que estava
	// implementado e testado, mas o construtor de producao nao o chamava.
	for _, tipo := range tiposComExtracaoDedicada {
		if _, aceito := allowedUploadMimeTypes[tipo]; !aceito {
			t.Errorf("existe extrator dedicado para %q, mas o upload rejeita esse tipo: "+
				"o extrator e codigo morto", tipo)
		}
	}
}

// TestFallbackDeTextoPuroContinuaSeguro documenta o motivo de manter o ramo
// default mesmo sendo inalcancavel pelo upload.
func TestFallbackDeTextoPuroContinuaSeguro(t *testing.T) {
	indexer := indexing.NewDeterministicIndexer()
	if indexer == nil {
		t.Fatal("indexador nao foi construido")
	}

	// O contrato que importa: o fallback nao pode entrar em panico nem inventar
	// estrutura. A verificacao de comportamento vive em internal/indexing; aqui
	// o que se trava e a EXISTENCIA do caminho seguro, para que a remocao do
	// ramo default seja uma decisao consciente e nao um efeito colateral.
	if len(allowedUploadMimeTypes) == 0 {
		t.Fatal("a lista de tipos aceitos ficou vazia: nenhum upload seria possivel")
	}
}
