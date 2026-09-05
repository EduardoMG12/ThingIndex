package usecase

import (
	"sort"
	"strings"

	"thing-index/internal/domain"
	"thing-index/internal/models"
)

// BuildNodeConnections deriva as conexoes de um no do sumario com nos de OUTROS
// documentos do acervo.
//
// # Por que a aresta e derivada e nao armazenada
//
// O principio II da constituicao do projeto permite grafos "apenas como trilha
// comparativa ou enriquecimento opcional -- nunca como dependencia do caminho
// principal". Uma tabela de arestas percorrida para recuperar seria um grafo e
// refutaria a tese do trabalho, que defende recuperacao relacional sem grafo
// como camada primaria.
//
// A saida desta funcao e recalculada a cada chamada, a partir do texto do
// proprio no, pelo mesmo scorer que ja atende as consultas de producao. Nada e
// persistido. A assinatura reflete isso de proposito: sem repositorio, sem
// contexto, sem erro -- e uma funcao pura, e violar a regra exigiria mudar a
// assinatura, o que nao passa despercebido em revisao.
//
// # Por que existe limiar
//
// A taxa de falso positivo medida nos seis bracos do experimento e 1,000:
// nenhum sabe responder "nao encontrei". Sem limiar, todo no teria conexao com
// algum outro e o canvas desenharia flecha em todo par. No ranking o falso
// positivo se esconde na quarta posicao e ninguem ve; numa aresta desenhada ele
// vira artefato visual apontado no projetor. `minScore` e o parametro que
// separa "existe evidencia" de "devolvi o menos ruim".
//
// # Por que a conexao carrega os termos
//
// Uma flecha sem justificativa e o mesmo tipo de resposta nao auditavel que
// este trabalho critica no RAG denso. MatchedTerms permite responder "esta
// conexao existe porque estes termos casaram", que e a versao da proveniencia
// aplicada a relacao entre trechos.
func BuildNodeConnections(
	origin models.DocumentOutlineNode,
	candidates []models.DocumentOutlineNode,
	minScore int,
) []domain.NodeConnection {
	queryTerms := tokenizeRetrievalText(outlineNodeSearchText(origin))
	if len(queryTerms) == 0 {
		return nil
	}

	// Palavras vazias saem ANTES da pontuacao, nao so da justificativa.
	//
	// Filtrar apenas na exibicao produziria o pior dos mundos: a conexao
	// continuaria existindo com score inflado por conectivos, mas apareceria
	// na tela sem nenhum termo que a explicasse -- uma aresta muda, que e
	// exatamente o tipo de resposta nao auditavel que este trabalho critica.
	significantTerms := make([]string, 0, len(queryTerms))
	for _, term := range queryTerms {
		if !ehPalavraVazia(term) {
			significantTerms = append(significantTerms, term)
		}
	}
	if len(significantTerms) == 0 {
		return nil
	}

	// Score zero nunca vira conexao, mesmo com minScore <= 0: aceitar zero
	// devolveria todo par do acervo como relacionado.
	threshold := minScore
	if threshold < 1 {
		threshold = 1
	}

	connections := make([]domain.NodeConnection, 0, len(candidates))

	for _, candidate := range candidates {
		// Conexao util liga documentos DIFERENTES. Trechos do mesmo arquivo
		// compartilham vocabulario por construcao e afogariam o sinal real.
		if candidate.DocumentID == origin.DocumentID {
			continue
		}

		text := outlineNodeSearchText(candidate)
		score := retrievalScore(text, significantTerms)
		if score < threshold {
			continue
		}

		connections = append(connections, domain.NodeConnection{
			DocumentID:   candidate.DocumentID,
			NodeID:       candidate.NodeID,
			Title:        candidate.Title,
			Summary:      candidate.Summary,
			StartPage:    candidate.StartPage,
			Score:        score,
			MatchedTerms: matchedTerms(text, significantTerms),
		})
	}

	// Ordem estavel e exigencia da tese: o trabalho denuncia o nao-determinismo
	// do indice aproximado, entao uma feature propria com ordem instavel seria
	// autorrefutacao. Score desc, depois NodeID asc para desempatar sem
	// depender da ordem de iteracao da entrada.
	sort.SliceStable(connections, func(i, j int) bool {
		if connections[i].Score != connections[j].Score {
			return connections[i].Score > connections[j].Score
		}
		return connections[i].NodeID < connections[j].NodeID
	})

	return connections
}

// palavrasVaziasPT sao termos que ligam qualquer par de documentos em portugues
// sem indicar relacao de conteudo.
//
// Existe por causa de um defeito observado no navegador: a tela exibiu uma
// conexao justificada por "1, de, do, servico". Artigos, preposicoes e
// numeracao de secao nao sao evidencia -- sao ruido que a contagem simples de
// termos promove a sinal.
//
// E a manifestacao visual do falso positivo de 1,000 medido nos seis bracos: no
// ranking o erro se esconde na quarta posicao, numa aresta desenhada ele fica
// exposto. A lista e curta e conservadora de proposito; ampliar demais correria
// o risco oposto, o de descartar termo com carga semantica no dominio
// (por exemplo "prazo", "valor", "parte").
var palavrasVaziasPT = map[string]struct{}{
	"a": {}, "ao": {}, "aos": {}, "as": {}, "com": {}, "como": {}, "da": {},
	"das": {}, "de": {}, "do": {}, "dos": {}, "e": {}, "em": {}, "entre": {},
	"na": {}, "nas": {}, "no": {}, "nos": {}, "o": {}, "os": {}, "ou": {},
	"para": {}, "pela": {}, "pelas": {}, "pelo": {}, "pelos": {}, "por": {},
	"que": {}, "se": {}, "sem": {}, "ser": {}, "sera": {}, "sobre": {},
	"um": {}, "uma": {}, "sao": {}, "foi": {}, "seu": {}, "sua": {},
}

// ehPalavraVazia diz se o termo e conectivo ou numeracao de secao.
//
// Numeros isolados ("1", "2") entram porque sao numeracao de titulo --
// "1 DO OBJETO" e "1 DA DISPONIBILIDADE" casariam pelo "1" sem qualquer
// relacao de conteudo. Numeros com mais de um digito ficam de fora: "99,5",
// "30" e "2025" costumam ser valores contratuais com significado real.
func ehPalavraVazia(termo string) bool {
	if len(termo) == 1 {
		return true
	}
	_, vazia := palavrasVaziasPT[termo]
	return vazia
}

// matchedTerms devolve, em ordem estavel, os termos da origem presentes no
// candidato -- descartando palavras vazias, que nao justificam nada.
func matchedTerms(text string, queryTerms []string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}

	terms := make([]string, 0, len(queryTerms))
	for _, term := range queryTerms {
		if ehPalavraVazia(term) {
			continue
		}
		if strings.Contains(normalized, term) {
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms
}
