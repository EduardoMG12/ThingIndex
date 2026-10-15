package indexing

import "testing"

// Um byte de controle dentro de uma <word> nao pode derrubar o documento
// inteiro (ex.: ESC, 0x1B).
func TestParsePDFBBoxLayoutTolerCaractereDeControle(t *testing.T) {
	raw := []byte(`<?xml version="1.0"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
<html xmlns="http://www.w3.org/1999/xhtml"><head><title></title></head><body>
<doc><page width="612" height="792"><flow><block xMin="72" yMin="80" xMax="300" yMax="94">
<line xMin="72" yMin="80" xMax="300" yMax="94"><word xMin="72" yMin="80" xMax="100" yMax="94">Grade</word><word xMin="110" yMin="80" xMax="114" yMax="94">` + "\x1b" + `</word><word xMin="120" yMin="80" xMax="300" yMax="94">Curricular</word></line>
</block></flow></page></doc></body></html>`)
	extraction, err := parsePDFBBoxLayout(raw)
	if err != nil {
		t.Fatalf("parse com ESC dentro de word: %v", err)
	}
	if extraction.PageCount != 1 {
		t.Fatalf("paginas = %d, quero 1", extraction.PageCount)
	}
}
