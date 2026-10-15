package usecase

import (
	"strconv"
	"testing"

	"thing-index/internal/models"
)

func noDeTeste(docID uint, nodeID, titulo, resumo string) models.DocumentOutlineNode {
	return models.DocumentOutlineNode{
		DocumentID: docID,
		NodeID:     nodeID,
		Title:      titulo,
		Summary:    resumo,
	}
}

func TestConexoesNaoApontamParaOProprioDocumento(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "Penalidades por atraso", "multa de 2 por cento sobre o valor da fatura em caso de atraso")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(1, "doc1#s2", "Outra secao do mesmo documento", "multa de 2 por cento sobre o valor da fatura tambem aqui"),
		noDeTeste(2, "doc2#s1", "Multa contratual", "a multa de 2 por cento sobre a fatura sera aplicada"),
	}

	conexoes := BuildNodeConnections(origem, nos, 1)

	for _, c := range conexoes {
		if c.DocumentID == origem.DocumentID {
			t.Fatalf("conexao aponta para o proprio documento: %s -> %s", origem.NodeID, c.NodeID)
		}
	}
	if len(conexoes) == 0 {
		t.Fatal("nenhuma conexao encontrada; o teste perdeu poder de deteccao")
	}
}

func TestConexoesSaoDeterministicas(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "Prazo de pagamento", "pagamento em ate trinta dias apos o aceite")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "Prazos", "o pagamento ocorre em trinta dias"),
		noDeTeste(3, "doc3#s1", "Aceite", "o aceite do servico precede o pagamento"),
		noDeTeste(4, "doc4#s1", "Trinta dias", "prazo de trinta dias para pagamento apos aceite"),
	}

	primeira := BuildNodeConnections(origem, nos, 1)
	for i := 0; i < 20; i++ {
		outra := BuildNodeConnections(origem, nos, 1)
		if len(outra) != len(primeira) {
			t.Fatalf("quantidade variou entre execucoes: %d vs %d", len(primeira), len(outra))
		}
		for j := range primeira {
			if primeira[j].NodeID != outra[j].NodeID || primeira[j].Score != outra[j].Score {
				t.Fatalf("ordem instavel na posicao %d: %s(%d) vs %s(%d)",
					j, primeira[j].NodeID, primeira[j].Score, outra[j].NodeID, outra[j].Score)
			}
		}
	}
}

// Sem limiar todo no teria conexao com algum outro.
func TestSemConexaoAcimaDoLimiarDevolveVazio(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "Cronograma de manutencao da frota", "revisao periodica dos veiculos da frota")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "Politica de privacidade", "tratamento de dados pessoais conforme a legislacao"),
		noDeTeste(3, "doc3#s1", "Disponibilidade", "uptime de 99,5 por cento medido mensalmente"),
	}

	// Limiar alto o bastante para nenhum candidato passar.
	conexoes := BuildNodeConnections(origem, nos, 99)

	if len(conexoes) != 0 {
		t.Fatalf("esperava lista vazia quando nada supera o limiar, veio %d conexoes: %+v",
			len(conexoes), conexoes)
	}
}

func TestLimiarZeroNaoLiberaTudo(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "Alfa", "conteudo totalmente distinto")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "Beta", "nenhuma palavra em comum"),
	}

	if conexoes := BuildNodeConnections(origem, nos, 0); len(conexoes) != 0 {
		t.Fatalf("score zero nao pode virar conexao; veio %+v", conexoes)
	}
}

func TestConexoesRespeitamOrdemDeScore(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "Multa por atraso na entrega", "multa por atraso na entrega do servico contratado")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "Fraca", "atraso"),
		noDeTeste(3, "doc3#s1", "Forte", "multa por atraso na entrega do servico contratado"),
	}

	conexoes := BuildNodeConnections(origem, nos, 1)
	if len(conexoes) < 2 {
		t.Fatalf("esperava ao menos 2 conexoes, veio %d", len(conexoes))
	}
	if conexoes[0].Score < conexoes[len(conexoes)-1].Score {
		t.Fatalf("conexoes fora de ordem: primeira=%d ultima=%d",
			conexoes[0].Score, conexoes[len(conexoes)-1].Score)
	}
	if conexoes[0].NodeID != "doc3#s1" {
		t.Fatalf("a conexao mais forte deveria ser doc3#s1, veio %s", conexoes[0].NodeID)
	}
}

func TestConexaoExplicaSeMesma(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "Penalidades", "multa de dois por cento sobre a fatura")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "Multa", "a multa incide sobre a fatura mensal"),
	}

	conexoes := BuildNodeConnections(origem, nos, 1)
	if len(conexoes) == 0 {
		t.Fatal("nenhuma conexao; sem poder de deteccao")
	}

	c := conexoes[0]
	if len(c.MatchedTerms) == 0 {
		t.Fatal("conexao sem termos que a justifiquem: nao e auditavel")
	}
	if c.Score <= 0 {
		t.Fatalf("conexao com score nao positivo: %d", c.Score)
	}
	// Os termos precisam existir na origem.
	origemTermos := map[string]bool{}
	for _, termo := range tokenizeRetrievalText(origem.Title + " " + origem.Summary) {
		origemTermos[termo] = true
	}
	for _, termo := range c.MatchedTerms {
		if !origemTermos[termo] {
			t.Fatalf("termo %q justifica a conexao mas nao esta na origem", termo)
		}
	}
}

// Conectivos e numeracao ("1, de, do") ligavam qualquer par de documentos.
func TestPalavrasVaziasNaoSustentamConexao(t *testing.T) {
	// So conectivos e numeracao em comum, nenhum substantivo.
	origem := noDeTeste(1, "doc1#s1", "1 DA GARANTIA", "o prazo e de trinta dias para a entrega")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "1 DO OBJETO", "a licitacao e para o fornecimento de equipamentos"),
	}

	conexoes := BuildNodeConnections(origem, nos, 2)

	if len(conexoes) != 0 {
		t.Fatalf("conexao criada apenas por conectivos: %+v", conexoes[0].MatchedTerms)
	}
}

func TestTermoSubstantivoAindaSustentaConexao(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "2 PENALIDADES", "multa de 5 por cento sobre a fatura mensal")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "2 DAS PENALIDADES", "a multa sera de 5 por cento sobre a fatura mensal"),
	}

	conexoes := BuildNodeConnections(origem, nos, 2)
	if len(conexoes) == 0 {
		t.Fatal("o filtro de palavras vazias matou uma conexao legitima")
	}

	temSubstantivo := false
	for _, termo := range conexoes[0].MatchedTerms {
		if termo == "multa" || termo == "fatura" || termo == "penalidades" || termo == "mensal" {
			temSubstantivo = true
		}
	}
	if !temSubstantivo {
		t.Fatalf("conexao legitima perdeu os termos que a explicam: %v", conexoes[0].MatchedTerms)
	}
}

func TestOrigemSemTextoNaoProduzConexao(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "", "")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "Qualquer coisa", "texto irrelevante"),
	}

	if conexoes := BuildNodeConnections(origem, nos, 1); len(conexoes) != 0 {
		t.Fatalf("no sem texto nao pode gerar conexao; veio %+v", conexoes)
	}
}

// Termo presente em quase todas as secoes nao e evidencia de relacao.
func TestTermoOnipresenteNaoSustentaConexao(t *testing.T) {
	origem := noDeTeste(1, "doc1#anexo-c", "ANEXO C DECLARACAO DE RENDA", "declaracao do curso do edital para o campus")
	nos := []models.DocumentOutlineNode{origem}
	// Trinta secoes de outros documentos, todas com "curso", "edital" e "campus".
	for i := 0; i < 30; i++ {
		nos = append(nos, noDeTeste(2, "doc2#s"+string(rune('a'+i%26))+string(rune('a'+i/26)), "SECAO", "o curso do edital no campus trata de outro assunto"))
	}
	// So um termo raro ("renda"), abaixo do limiar 2.
	nos = append(nos, noDeTeste(3, "doc3#renda", "COMPROVANTE DE RENDA", "renda familiar"))
	conexoes := BuildNodeConnections(origem, nos, 2)
	if len(conexoes) != 0 {
		t.Fatalf("termos onipresentes sustentaram conexao: %+v", conexoes)
	}
	// Dois termos raros em comum sustentam.
	nos = append(nos, noDeTeste(4, "doc4#renda-declaracao", "DECLARACAO DE RENDA", "declaracao de renda familiar"))
	conexoes = BuildNodeConnections(origem, nos, 2)
	if len(conexoes) != 1 || conexoes[0].NodeID != "doc4#renda-declaracao" {
		t.Fatalf("termos raros deveriam sustentar exatamente uma conexao: %+v", conexoes)
	}
	for _, termo := range conexoes[0].MatchedTerms {
		if termo == "curso" || termo == "edital" || termo == "campus" {
			t.Fatalf("termo onipresente na justificativa: %v", conexoes[0].MatchedTerms)
		}
	}
}

// Por substring, "lei" casava "leitura" e "estagio" casava "estagiario". Por
// token inteiro, nao.
func TestSubstringNaoCasa(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "DA LEI", "a lei do estagio manda")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "LEITURA", "a leitura do estagiario manda"),
		noDeTeste(3, "doc3#s1", "OUTRO", "outro assunto"),
		noDeTeste(4, "doc4#s1", "MAIS UM", "mais um assunto"),
	}
	if conexoes := BuildNodeConnections(origem, nos, 2); len(conexoes) != 0 {
		t.Fatalf("pedaco de palavra sustentou conexao: %+v", conexoes)
	}
}

func TestNumeroENomeProprioNaoSustentamConexao(t *testing.T) {
	origem := noDeTeste(1, "doc1#s1", "MEMBROS", "Andreia Marini assina em 2022 a portaria 18 sobre laboratorio")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "QUADRO DO TECNICO", "Andreia Marini apresenta em 2022 o corpo docente"),
		noDeTeste(3, "doc3#s1", "OUTRO", "outro assunto"),
		noDeTeste(4, "doc4#s1", "MAIS UM", "mais um assunto"),
	}
	if conexoes := BuildNodeConnections(origem, nos, 2); len(conexoes) != 0 {
		t.Fatalf("nome proprio e ano sustentaram conexao: %+v", conexoes)
	}
}

func TestMesmaNormaSustentaConexaoTipada(t *testing.T) {
	origem := noDeTeste(1, "doc1#estagio", "DO ESTAGIO", "regido pela Lei nº 11.788/2008 e pela Resolução CONSUP 82/2022")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#politica", "POLITICA DE ESTAGIOS", "esta resolucao 82/2022 dispoe sobre convenios"),
		noDeTeste(3, "doc3#outro", "OUTRO", "resolucao 50/2017 sobre avaliacao"),
		noDeTeste(4, "doc4#mais", "MAIS UM", "mais um assunto"),
	}
	conexoes := BuildNodeConnections(origem, nos, 2)
	if len(conexoes) != 1 || conexoes[0].NodeID != "doc2#politica" || conexoes[0].Kind != NodeConnectionKindSameNorm {
		t.Fatalf("esperava uma conexao mesma-norma com doc2: %+v", conexoes)
	}
	if conexoes[0].MatchedTerms[0] != "resolucao 82/2022" {
		t.Fatalf("a citacao normalizada deveria justificar a aresta: %v", conexoes[0].MatchedTerms)
	}
}

func TestTetoDeGrauCortaOHub(t *testing.T) {
	origem := noDeTeste(1, "doc1#hub", "HUB", "zebra quokka")
	nos := []models.DocumentOutlineNode{origem}
	for i := 0; i < 30; i++ {
		nos = append(nos, noDeTeste(uint(2+i), "doc#"+string(rune('a'+i%26))+string(rune('a'+i/26)), "S", "zebra quokka aqui"))
	}
	// Com 30 candidatos o piso seria 2; sobe o numero para os termos contarem
	// como raros.
	for i := 0; i < 1500; i++ {
		nos = append(nos, noDeTeste(9999, "enche#"+strconv.Itoa(i), "E", "assunto qualquer"))
	}
	conexoes := BuildNodeConnections(origem, nos, 2)
	if len(conexoes) != NodeConnectionMaxDegree {
		t.Fatalf("esperava o teto de %d arestas, veio %d", NodeConnectionMaxDegree, len(conexoes))
	}
}

func TestNumeralRomanoEAparatoBibliograficoNaoSustentamConexao(t *testing.T) {
	origem := noDeTeste(1, "doc1#cap", "CAPITULO VI", "isbn 9788577803828 ed. Bookman vi")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#cap", "CAPITULO VI", "isbn 9788535201580 ed. Campus vi"),
		noDeTeste(3, "doc3#s", "OUTRO", "outro assunto"),
		noDeTeste(4, "doc4#s", "MAIS", "mais um assunto"),
	}
	if conexoes := BuildNodeConnections(origem, nos, 2); len(conexoes) != 0 {
		t.Fatalf("'vi', 'isbn' e 'ed' sustentaram conexao: %+v", conexoes)
	}
}

// Dois trechos com o mesmo codigo de disciplina (SIS07302) se conectam por
// isso, com o tipo "mesma-disciplina", mesmo sem termo raro em comum.
func TestConexaoPorCodigoDeDisciplina(t *testing.T) {
	t.Parallel()
	origem := models.DocumentOutlineNode{DocumentID: 1, NodeID: "matriz#3", Title: "Terceiro periodo", Metadata: map[string]any{"retrieval_text": "SIS07302 Estrutura de Dados I 60 horas"}}
	candidatos := []models.DocumentOutlineNode{
		{DocumentID: 2, NodeID: "ementas#12", Title: "Ementa", Metadata: map[string]any{"retrieval_text": "Componente SIS07302: listas, pilhas, filas e arvores."}},
		{DocumentID: 2, NodeID: "ementas#13", Title: "Outra ementa", Metadata: map[string]any{"retrieval_text": "Componente SIS07405: redes de computadores."}},
	}
	conexoes := BuildNodeConnections(origem, candidatos, 2)
	if len(conexoes) != 1 || conexoes[0].NodeID != "ementas#12" || conexoes[0].Kind != NodeConnectionKindSameDiscipline {
		t.Fatalf("esperava uma conexao mesma-disciplina com ementas#12: %+v", conexoes)
	}
	if conexoes[0].MatchedTerms[0] != "disciplina SIS07302" {
		t.Fatalf("a justificativa devia ser o codigo: %v", conexoes[0].MatchedTerms)
	}
}
