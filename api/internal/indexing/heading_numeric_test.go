package indexing

import "testing"

// Linhas de calendario e de tabela de datas nao sao titulos de secao, mesmo
// comecando com numero (o que casa com numberedHeadingPattern).
func TestLooksLikeHeadingRejeitaLinhaDominadaPorNumeros(t *testing.T) {
	rejeitar := []string{
		"1 2 3 1 Feriado Nacional",
		"22 23 24 25 26 27 28",
		"4 5 6 7 8 9 10",
		"9 a 13 Confirmacao",
		"1 2 3 1 Feriado Nacional - Confraternizacao Universal",
		"15 16 17 18 19 20 21 2 a 6 Rematricula dos estudantes veteranos - Ensino Superior 17 Carnaval",
		"14 dias letivos 6 23 Prazo de entrega dos Planos de Ensino para 2026-1",
	}
	for _, linha := range rejeitar {
		if looksLikeHeading(linha) {
			t.Errorf("%q virou titulo de secao; e linha de calendario", linha)
		}
	}
	aceitar := []string{
		"1 DAS DISPOSICOES PRELIMINARES",
		"2.3 Da inscricao",
		"ANEXO II",
		"3 DISPOSITIVO",
	}
	for _, linha := range aceitar {
		if !looksLikeHeading(linha) {
			t.Errorf("%q deixou de ser titulo de secao", linha)
		}
	}
}
