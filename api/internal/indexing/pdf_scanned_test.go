package indexing

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"

	"thing-index/internal/models"
)

// PDF de 3 paginas sem nenhuma camada de texto (so um traco vetorial por
// pagina), equivalente a um processo digitalizado.
const scannedPDFBase64 = `JVBERi0xLjQKMSAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZyAvUGFnZXMgMiAwIFIgPj4KZW5kb2Jq
CjIgMCBvYmoKPDwgL1R5cGUgL1BhZ2VzIC9LaWRzIFszIDAgUiA1IDAgUiA3IDAgUl0gL0NvdW50
IDMgPj4KZW5kb2JqCjMgMCBvYmoKPDwgL1R5cGUgL1BhZ2UgL1BhcmVudCAyIDAgUiAvTWVkaWFC
b3ggWzAgMCA2MTIgNzkyXSAvQ29udGVudHMgNCAwIFIgPj4KZW5kb2JqCjQgMCBvYmoKPDwgL0xl
bmd0aCAzMCA+PgpzdHJlYW0KMCAwIDAgUkcgMTAwIDEwMCBtIDIwMCAyMDAgbCBTCmVuZHN0cmVh
bQplbmRvYmoKNSAwIG9iago8PCAvVHlwZSAvUGFnZSAvUGFyZW50IDIgMCBSIC9NZWRpYUJveCBb
MCAwIDYxMiA3OTJdIC9Db250ZW50cyA2IDAgUiA+PgplbmRvYmoKNiAwIG9iago8PCAvTGVuZ3Ro
IDMwID4+CnN0cmVhbQowIDAgMCBSRyAxMDAgMTAwIG0gMjAwIDIwMCBsIFMKZW5kc3RyZWFtCmVu
ZG9iago3IDAgb2JqCjw8IC9UeXBlIC9QYWdlIC9QYXJlbnQgMiAwIFIgL01lZGlhQm94IFswIDAg
NjEyIDc5Ml0gL0NvbnRlbnRzIDggMCBSID4+CmVuZG9iago4IDAgb2JqCjw8IC9MZW5ndGggMzAg
Pj4Kc3RyZWFtCjAgMCAwIFJHIDEwMCAxMDAgbSAyMDAgMjAwIGwgUwplbmRzdHJlYW0KZW5kb2Jq
CnhyZWYKMCA5CjAwMDAwMDAwMDAgNjU1MzUgZiAKMDAwMDAwMDAwOSAwMDAwMCBuIAowMDAwMDAw
MDU4IDAwMDAwIG4gCjAwMDAwMDAxMjcgMDAwMDAgbiAKMDAwMDAwMDIxNCAwMDAwMCBuIAowMDAw
MDAwMjk0IDAwMDAwIG4gCjAwMDAwMDAzODEgMDAwMDAgbiAKMDAwMDAwMDQ2MSAwMDAwMCBuIAow
MDAwMDAwNTQ4IDAwMDAwIG4gCnRyYWlsZXIKPDwgL1NpemUgOSAvUm9vdCAxIDAgUiA+PgpzdGFy
dHhyZWYKNjI4CiUlRU9GCg==`

func scannedPDFBytes(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(stripWhitespace(scannedPDFBase64))
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return data
}

func stripWhitespace(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		if r != '\n' && r != '\r' && r != '\t' && r != ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Regressao: PDF totalmente digitalizado abortava com "implausible text"
// antes do fallback de OCR.
func TestScannedPDFReachesOCRFallback(t *testing.T) {
	t.Parallel()

	extractor := newCommandPDFTextExtractor()
	if extractor == nil {
		t.Skip("pdftotext indisponivel no PATH")
	}

	ocr := &fakePDFOCRExtractor{texts: map[int]string{
		1: "JUIZO DE DIREITO DA VARA CIVEL. Processo 0001234-56.2024.8.16.0000 distribuido em 2024.",
		2: "A parte autora requer a procedencia do pedido e a condenacao da re em custas processuais.",
		3: "Valor da causa: R$ 50.000,00. Comarca de Palmas, Estado do Parana, Republica Federativa.",
	}}

	indexer := newDeterministicIndexerWithExtractors(extractor, ocr)
	doc := &models.Document{ID: 1, MimeType: "application/pdf"}

	result, err := indexer.Index(context.Background(), doc, bytes.NewReader(scannedPDFBytes(t)))
	if err != nil {
		t.Fatalf("indexacao falhou antes do OCR: %v", err)
	}

	if len(ocr.calls) == 0 {
		t.Fatal("o extrator de OCR nunca foi chamado")
	}
	// Todas as tres paginas precisam ser oferecidas ao OCR, nao so a primeira.
	if got := ocr.calls[0]; len(got) != 3 {
		t.Fatalf("OCR recebeu as paginas %v, quero as 3", got)
	}
	if len(result.ParagraphNodes) == 0 {
		t.Fatal("nenhum paragrafo extraido apos o OCR")
	}
}
