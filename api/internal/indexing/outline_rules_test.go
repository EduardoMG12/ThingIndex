package indexing

import "testing"

// Frase solta nao e titulo de arquetipo num documento numerado; bibliografia
// numerada nao e pagina de sumario.
func TestFraseSoltaNaoViraCapituloEBibliografiaNaoEhSumario(t *testing.T) {
	t.Parallel()
	if numberedOrArchetypeHeading("decisão junto às/aos discentes, baseadas nas normas deste PPC.") {
		t.Fatal("frase que comeca com minuscula nao pode ser titulo")
	}
	if numberedOrArchetypeHeading("30 Horas SERGIO MERLIN Graduação em Engenharia da (Jornada Técnico em Informática BAU JUNIOR computação Flexibilizada)") {
		t.Fatal("linha de tabela com 16 palavras nao e capitulo")
	}
	if !numberedOrArchetypeHeading("8 INFRAESTRUTURA") || !numberedOrArchetypeHeading("ANEXO A") {
		t.Fatal("titulo numerado e arquetipo continuam valendo")
	}
	bibliografia := textBlock{Content: "1. DEITEL, Paul J. Java: como programar. 10. ed. São Paulo: Pearson, 2016.\n2. ELMASRI, Ramez. Sistemas de banco de dados. 6. ed. São Paulo: Pearson, 2011.\n3. TANENBAUM, Andrew. Redes de computadores. 5. ed. São Paulo: Pearson, 2011.\n4. SOMMERVILLE, Ian. Engenharia de software. 9. ed. São Paulo: Pearson, 2011.\n5. PRESSMAN, Roger. Engenharia de software. 7. ed. Porto Alegre: AMGH, 2011.\n6. SILBERSCHATZ, Abraham. Sistemas operacionais. 9. ed. Rio de Janeiro: LTC, 2015."}
	for _, l := range []string{"1. DEITEL, Paul J. Java: como programar. 10. ed. São Paulo: Pearson, 2016.", "2. ELMASRI, Ramez. Sistemas de banco de dados. 6. ed. São Paulo: Pearson, 2011.", "3. TANENBAUM, Andrew. Redes de computadores. 5. ed. São Paulo: Pearson, 2011.", "4. SOMMERVILLE, Ian. Engenharia de software. 9. ed. São Paulo: Pearson, 2011.", "5. PRESSMAN, Roger. Engenharia de software. 7. ed. Porto Alegre: AMGH, 2011.", "6. SILBERSCHATZ, Abraham. Sistemas operacionais. 9. ed. Rio de Janeiro: LTC, 2015."} {
		bibliografia.Lines = append(bibliografia.Lines, textLine{Text: l})
	}
	if isPDFTableOfContentsPage([]textBlock{bibliografia}) {
		t.Fatal("bibliografia numerada nao e pagina de sumario")
	}
	sumario := textBlock{Content: "1 APRESENTAÇÃO\n1.1 IDENTIFICAÇÃO\n1.2 HISTÓRICO\n2 JUSTIFICATIVA\n3 OBJETIVOS"}
	for _, l := range []string{"1 APRESENTAÇÃO", "1.1 IDENTIFICAÇÃO", "1.2 HISTÓRICO", "2 JUSTIFICATIVA", "3 OBJETIVOS"} {
		sumario.Lines = append(sumario.Lines, textLine{Text: l})
	}
	if !isPDFTableOfContentsPage([]textBlock{sumario}) {
		t.Fatal("bloco de sumario em duas colunas continua sendo sumario")
	}
}
