package usecase

import (
	"strconv"
	"testing"

	"thing-index/internal/models"
)

// Conexoes entre trechos de documentos diferentes.
//
// A regra que governa este arquivo vem do principio II da constituicao:
// "embeddings, vetores e GRAFOS sao permitidos apenas como trilha comparativa
// ou enriquecimento opcional -- nunca como dependencia do caminho principal".
//
// Uma aresta ARMAZENADA (tabela de links percorrida para recuperar) e um grafo
// e refuta a tese do trabalho. Uma aresta DERIVADA -- recalculada a cada
// chamada, a partir do texto do proprio no, pelo mesmo scorer relacional que ja
// existe -- e apenas a saida do recuperador desenhada na tela.
//
// Por isso a funcao nao recebe repositorio, nao escreve nada e e pura: a
// assinatura torna a violacao dificil de cometer sem perceber.

func noDeTeste(docID uint, nodeID, titulo, resumo string) models.DocumentOutlineNode {
	return models.DocumentOutlineNode{
		DocumentID: docID,
		NodeID:     nodeID,
		Title:      titulo,
		Summary:    resumo,
	}
}

// TestConexoesNaoApontamParaOProprioDocumento trava a regra mais basica: uma
// conexao util liga documentos DIFERENTES. Trechos do mesmo arquivo compartilham
// vocabulario por construcao e produziriam ruido que afogaria o sinal real.
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

// TestConexoesSaoDeterministicas e exigencia direta da tese: o trabalho denuncia
// o nao-determinismo do indice vetorial aproximado. Uma feature propria que
// devolvesse ordem instavel seria autorrefutacao.
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

// TestSemConexaoAcimaDoLimiarDevolveVazio e a tarefa 4.5 da spec, e responde ao
// achado mais grave do trabalho: taxa de falso positivo 1,000, nenhum braco sabe
// dizer "nao encontrei".
//
// Sem limiar, TODO no teria conexao com algum outro, e o canvas desenharia
// flecha em todo par. No ranking o falso positivo se esconde na quarta posicao;
// no canvas ele vira artefato visual apontado no projetor.
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

// TestLimiarZeroNaoLiberaTudo protege contra o erro classico de limiar: aceitar
// score zero devolveria todo par do acervo como conexao.
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

// TestConexoesRespeitamOrdemDeScore garante que a mais forte vem primeiro: e o
// que o usuario le como "a flecha mais grossa".
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

// TestConexaoExplicaSeMesma trava a exigencia de auditabilidade: a aresta
// precisa dizer POR QUE existe. Uma flecha sem justificativa e exatamente o
// tipo de resposta nao auditavel que este trabalho critica no RAG denso.
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
	// Os termos precisam existir de fato na origem -- senao a justificativa
	// seria decorativa.
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

// TestPalavrasVaziasNaoSustentamConexao trava o defeito visto no navegador: a
// tela exibiu uma conexao justificada por "1, de, do, servico" -- artigos e
// preposicoes que ligam qualquer par de documentos em portugues.
//
// Isto e o falso positivo de 1,000 se manifestando na camada visual. No ranking
// ele se esconde na quarta posicao; numa aresta desenhada vira artefato que o
// avaliador aponta no projetor. Termo vazio nao e evidencia de relacao.
func TestPalavrasVaziasNaoSustentamConexao(t *testing.T) {
	// A origem compartilha com o candidato APENAS conectivos e a numeracao de
	// secao. Nenhum substantivo em comum -- se houvesse, a conexao seria
	// legitima e o teste nao provaria nada sobre o filtro.
	origem := noDeTeste(1, "doc1#s1", "1 DA GARANTIA", "o prazo e de trinta dias para a entrega")
	nos := []models.DocumentOutlineNode{
		origem,
		noDeTeste(2, "doc2#s1", "1 DO OBJETO", "a licitacao e para o fornecimento de equipamentos"),
	}

	conexoes := BuildNodeConnections(origem, nos, 2)

	// Sem termos significativos em comum, nao pode haver conexao nenhuma.
	if len(conexoes) != 0 {
		t.Fatalf("conexao criada apenas por conectivos: %+v", conexoes[0].MatchedTerms)
	}
}

// TestTermoSubstantivoAindaSustentaConexao garante que o filtro nao esvaziou o
// sinal legitimo -- remover palavras vazias nao pode matar a feature.
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

// Termo presente em quase todas as secoes do acervo ("curso", "edital") nao
// e evidencia de relacao: e o vocabulario do acervo. Com ele contando, duas
// declaracoes de formulario sem nada em comum apareciam ligadas por "2 termos".
func TestTermoOnipresenteNaoSustentaConexao(t *testing.T) {
	origem := noDeTeste(1, "doc1#anexo-c", "ANEXO C DECLARACAO DE RENDA", "declaracao do curso do edital para o campus")
	nos := []models.DocumentOutlineNode{origem}
	// Trinta secoes de outros documentos, todas com "curso", "edital" e "campus".
	for i := 0; i < 30; i++ {
		nos = append(nos, noDeTeste(2, "doc2#s"+string(rune('a'+i%26))+string(rune('a'+i/26)), "SECAO", "o curso do edital no campus trata de outro assunto"))
	}
	// So esta compartilha um termo raro ("renda") -- e apenas um, abaixo do limiar 2.
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
	// 30 candidatos: 2% = 0,6 -> piso 2... entao "zebra" e "quokka" nao seriam
	// raros. Sobe o piso com candidatos suficientes para os termos contarem.
	for i := 0; i < 1500; i++ {
		nos = append(nos, noDeTeste(9999, "enche#"+strconv.Itoa(i), "E", "assunto qualquer"))
	}
	conexoes := BuildNodeConnections(origem, nos, 2)
	if len(conexoes) != NodeConnectionMaxDegree {
		t.Fatalf("esperava o teto de %d arestas, veio %d", NodeConnectionMaxDegree, len(conexoes))
	}
}
