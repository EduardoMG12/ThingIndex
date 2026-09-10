package indexing

import "testing"

// A pagina de sumario do PPC ("1.1.1 Denominacao do Curso 9", 124 linhas)
// duplicava cada secao do documento. Cinco ou mais linhas terminadas em
// numero de pagina marcam a pagina como sumario, e nela nada vira titulo.
func TestPaginaDeSumarioNaoGeraTitulos(t *testing.T) {
	page := 3
	blocks := []textBlock{
		{Content: "1. APRESENTAÇÃO DO PROJETO 9", PageNumber: &page},
		{Content: "1.1 IDENTIFICAÇÃO 9", PageNumber: &page},
		{Content: "1.1.1 Denominação do Curso 9", PageNumber: &page},
		{Content: "1.1.2 Área do Conhecimento 9", PageNumber: &page},
		{Content: "2. JUSTIFICATIVA 12", PageNumber: &page},
		{Content: "APÊNDICE A - REGULAMENTO DE ESTÁGIO 176", PageNumber: &page},
	}
	if !isPDFTableOfContentsPage(blocks) {
		t.Fatal("seis entradas com numero de pagina deveriam marcar a pagina como sumario")
	}
	corpo := 9
	pages := map[int][]textBlock{
		3: blocks,
		9: {
			{Content: "1. APRESENTAÇÃO DO PROJETO", PageNumber: &corpo},
			{Content: "O curso foi criado em 2010 e atende a regiao sudoeste.", PageNumber: &corpo},
		},
	}
	headings, _ := buildPDFStructureFromPageBlocks(pages, 9)
	if len(headings) != 1 || headings[0].Title != "1. APRESENTAÇÃO DO PROJETO" || headings[0].PageNumber == nil || *headings[0].PageNumber != 9 {
		t.Fatalf("esperava so o titulo do corpo (p. 9), veio %+v", headings)
	}
	// Quatro entradas nao bastam: uma lista curta de itens numerados nao e sumario.
	if isPDFTableOfContentsPage(blocks[:4]) {
		t.Fatal("quatro entradas nao deveriam marcar sumario")
	}
}

// "Teorica 67", "Total 205", "Subtotal 4 80 40": celulas de tabela em caixa
// mista com numero solto viravam secao (33 de 50 na matriz curricular).
// Titulo numerado ("2.1 Objetivos") e caixa alta ("ANEXO X - C") continuam.
func TestCelulaDeTabelaComNumeroNaoEhTitulo(t *testing.T) {
	// Tambem: valor de campo de uma palavra, item terminado em ponto e virgula
	// e pedaco de titulo quebrado que comeca por preposicao.
	for _, celula := range []string{"Teórica 67", "Total 205", "Subtotal 4 80 40", "Período Disciplina 3", "III. Ética;", "20/06/2002;", "DE ENSINO", "E CONSUMO", "Programação de", "Tecnologia da", "29 de dezembro de 2008, que institui a Rede Federal"} {
		if looksLikeHeading(celula) {
			t.Fatalf("%q nao deveria ser titulo", celula)
		}
	}
	// So em PDF: valor de campo de uma palavra. Em texto puro "Termination"
	// sozinho numa linha continua sendo titulo.
	for _, valor := range []string{"Reitor", "Presencial", "Bacharelado"} {
		if looksLikePDFHeading(valor) {
			t.Fatalf("%q nao deveria ser titulo em PDF", valor)
		}
	}
	if !looksLikeHeading("Termination") {
		t.Fatal("uma palavra em texto puro continua podendo ser titulo")
	}
	for _, titulo := range []string{"2.1 Objetivos Específicos", "ANEXO X - C", "1 DAS DISPOSIÇÕES PRELIMINARES", "Metodologia da Pesquisa", "DAS DISPOSIÇÕES GERAIS", "DO CONSELHO SUPERIOR"} {
		if !looksLikeHeading(titulo) {
			t.Fatalf("%q deveria continuar sendo titulo", titulo)
		}
	}
}
