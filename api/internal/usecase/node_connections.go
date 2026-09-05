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
		score := retrievalScore(text, queryTerms)
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
			MatchedTerms: matchedTerms(text, queryTerms),
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

// matchedTerms devolve, em ordem estavel, os termos da origem presentes no
// candidato. E a justificativa auditavel da aresta.
func matchedTerms(text string, queryTerms []string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}

	terms := make([]string, 0, len(queryTerms))
	for _, term := range queryTerms {
		if strings.Contains(normalized, term) {
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms
}
