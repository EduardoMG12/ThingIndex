package usecase

// Motor de recuperacao da colecao e do documento (parecer B1): tokenizacao,
// expansao numerica, BM25 e pesos por tipo de fonte. Movido de
// document_usecase.go sem alteracao de codigo; o comportamento e travado pelos
// testes do pacote e pela saida de cmd/indexing-benchmark.

import (
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"thing-index/internal/domain"
	"thing-index/internal/indexing"
	"thing-index/internal/models"
	"thing-index/internal/retrieval"
	"unicode"
)

// buildTreeRetrieval usa as opcoes padrao; ver buildCollectionRetrieval.
func buildTreeRetrieval(query string, nodes []models.DocumentOutlineNode, pages []models.DocumentPage, chunks []models.DocumentChunk) *domain.TreeRetrievalResult {
	return buildTreeRetrievalWith(DefaultRetrievalOptions(), query, nodes, pages, chunks)
}

func buildTreeRetrievalWith(opts RetrievalOptions, query string, nodes []models.DocumentOutlineNode, pages []models.DocumentPage, chunks []models.DocumentChunk) *domain.TreeRetrievalResult {
	pageFields := opts.pageSearchFields()
	queryTerms := tokenizeRetrievalText(query)
	if len(queryTerms) == 0 {
		return nil
	}

	directEvidence := buildTreeDirectEvidence(chunks, queryTerms)
	baseNodeHits, _ := traverseTree(queryTerms, nodes, nil)
	nodeHits, traversalSteps := traverseTree(queryTerms, nodes, directEvidence.NodeScoreByID)
	pageScoreByNumber := make(map[int]int, len(pages))
	for _, step := range traversalSteps {
		if step.StartPage == nil || step.EndPage == nil {
			continue
		}
		for pageNumber := *step.StartPage; pageNumber <= *step.EndPage; pageNumber++ {
			pageScoreByNumber[pageNumber] += step.Score
		}
	}

	bestDirectPageScore := 0
	pageHits := make([]domain.TreePageHit, 0, len(pages))
	for _, page := range pages {
		directScore := scoreTreePage(page, queryTerms, pageFields) + directEvidence.PageScoreByNumber[page.PageNumber]
		if directScore > bestDirectPageScore {
			bestDirectPageScore = directScore
		}

		score := pageScoreByNumber[page.PageNumber] + directScore
		if score == 0 {
			continue
		}

		pageHits = append(pageHits, domain.TreePageHit{
			PageNumber: page.PageNumber,
			Summary:    page.Summary,
			Snippet:    clampRetrievalSnippet(page.Content, 220),
			Score:      score,
		})
	}

	sort.Slice(pageHits, func(i, j int) bool {
		if pageHits[i].Score == pageHits[j].Score {
			return pageHits[i].PageNumber < pageHits[j].PageNumber
		}
		return pageHits[i].Score > pageHits[j].Score
	})
	if len(pageHits) > 5 {
		pageHits = pageHits[:5]
	}

	strategy := "tree-iterative"
	if shouldPreferDirectTreeEvidence(baseNodeHits, nodeHits, nodes, directEvidence, bestDirectPageScore) {
		strategy = "tree-degraded-direct-evidence"
	} else if len(nodeHits) == 0 && bestDirectPageScore > 0 {
		strategy = "tree-direct-evidence"
	}

	return &domain.TreeRetrievalResult{
		Query:          query,
		NodeHits:       nodeHits,
		PageHits:       pageHits,
		TraversalSteps: traversalSteps,
		UsedTree:       true,
		Strategy:       strategy,
	}
}

type treeDirectEvidence struct {
	NodeScoreByID     map[string]int
	PageScoreByNumber map[int]int
	MaxNodeScore      int
	MaxPageScore      int
}

func buildTreeDirectEvidence(chunks []models.DocumentChunk, queryTerms []string) treeDirectEvidence {
	evidence := treeDirectEvidence{
		NodeScoreByID:     make(map[string]int),
		PageScoreByNumber: make(map[int]int),
	}

	for _, chunk := range chunks {
		score := scoreTreeChunk(chunk, queryTerms)
		if score == 0 {
			continue
		}

		if score > evidence.MaxPageScore {
			evidence.MaxPageScore = score
		}

		if nodeID := metadataStringValue(chunk.Metadata, "outline_node_id"); nodeID != "" {
			evidence.NodeScoreByID[nodeID] += score
			if evidence.NodeScoreByID[nodeID] > evidence.MaxNodeScore {
				evidence.MaxNodeScore = evidence.NodeScoreByID[nodeID]
			}
		}

		startPage, endPage, ok := chunkPageRange(chunk)
		if !ok {
			continue
		}
		for pageNumber := startPage; pageNumber <= endPage; pageNumber++ {
			evidence.PageScoreByNumber[pageNumber] += score
			if evidence.PageScoreByNumber[pageNumber] > evidence.MaxPageScore {
				evidence.MaxPageScore = evidence.PageScoreByNumber[pageNumber]
			}
		}
	}

	return evidence
}

func traverseTree(queryTerms []string, nodes []models.DocumentOutlineNode, directNodeScores map[string]int) ([]domain.TreeNodeHit, []domain.TreeTraversalStep) {
	nodeByID := make(map[string]models.DocumentOutlineNode, len(nodes))
	parentByNodeID := make(map[string]string, len(nodes))
	allHits := rankNodesForTraversal(nodes, queryTerms, directNodeScores)
	if len(allHits) == 0 {
		return nil, nil
	}

	scoreByNodeID := make(map[string]domain.TreeNodeHit, len(allHits))
	for _, node := range nodes {
		nodeByID[node.NodeID] = node
		if parentNodeID, ok := node.Metadata["parent_node_id"].(string); ok && strings.TrimSpace(parentNodeID) != "" {
			parentByNodeID[node.NodeID] = parentNodeID
		}
	}
	for _, hit := range allHits {
		scoreByNodeID[hit.NodeID] = hit
	}

	steps := buildTraversalSteps(allHits[0], nodeByID, parentByNodeID, scoreByNodeID)
	if len(allHits) > 5 {
		allHits = allHits[:5]
	}

	return allHits, steps
}

func rankNodesForTraversal(nodes []models.DocumentOutlineNode, queryTerms []string, directNodeScores map[string]int) []domain.TreeNodeHit {
	hits := make([]domain.TreeNodeHit, 0, len(nodes))
	for _, node := range nodes {
		score := retrievalScore(outlineNodeSearchText(node), queryTerms)
		score += metadataIntValue(node.Metadata, "confidence_score") / 25
		directScore := directNodeScores[node.NodeID]
		if directScore > 0 {
			score += directScore
			score += 2
		}
		if isWeakTreeNode(node, queryTerms, score) && directScore == 0 {
			score = max(0, score-2)
		}
		if score == 0 {
			continue
		}
		hits = append(hits, domain.TreeNodeHit{
			NodeID:        node.NodeID,
			Title:         node.Title,
			Summary:       node.Summary,
			Level:         node.Level,
			StartPage:     node.StartPage,
			EndPage:       node.EndPage,
			Score:         score,
			SequenceOrder: node.SequenceOrder,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			if hits[i].Level == hits[j].Level {
				return hits[i].SequenceOrder < hits[j].SequenceOrder
			}
			return hits[i].Level > hits[j].Level
		}
		return hits[i].Score > hits[j].Score
	})

	return hits
}

func shouldPreferDirectTreeEvidence(
	baseNodeHits []domain.TreeNodeHit,
	nodeHits []domain.TreeNodeHit,
	nodes []models.DocumentOutlineNode,
	directEvidence treeDirectEvidence,
	bestDirectPageScore int,
) bool {
	if len(nodeHits) == 0 {
		return false
	}

	nodeByID := make(map[string]models.DocumentOutlineNode, len(nodes))
	for _, node := range nodes {
		nodeByID[node.NodeID] = node
	}

	if len(baseNodeHits) == 0 {
		return directEvidence.MaxNodeScore > 0 || bestDirectPageScore > 0
	}

	baseTop := baseNodeHits[0]
	baseTopNode := nodeByID[baseTop.NodeID]
	if !isWeakTreeNode(baseTopNode, nil, baseTop.Score) {
		return false
	}

	if directEvidence.MaxNodeScore == 0 && bestDirectPageScore == 0 {
		return false
	}

	if nodeHits[0].NodeID != baseTop.NodeID {
		return true
	}

	return bestDirectPageScore >= baseTop.Score || directEvidence.MaxNodeScore >= baseTop.Score
}

func isWeakTreeNode(node models.DocumentOutlineNode, queryTerms []string, score int) bool {
	if fallbackStructure, ok := node.Metadata["fallback_structure"].(bool); ok && fallbackStructure {
		return true
	}
	if metadataStringValue(node.Metadata, "segment_type") == "page" {
		return true
	}
	if confidenceBand := metadataStringValue(node.Metadata, "confidence_band"); confidenceBand == "low" {
		return true
	}
	confidenceScore := metadataIntValue(node.Metadata, "confidence_score")
	if confidenceScore > 0 && confidenceScore < 60 {
		return true
	}

	minScore := 2
	if queryTermCount := len(queryTerms); queryTermCount >= 4 {
		minScore = 3
	} else if queryTermCount == 0 {
		minScore = 1
	}

	return score < minScore
}

func buildTraversalSteps(
	best domain.TreeNodeHit,
	nodeByID map[string]models.DocumentOutlineNode,
	parentByNodeID map[string]string,
	scoreByNodeID map[string]domain.TreeNodeHit,
) []domain.TreeTraversalStep {
	if strings.TrimSpace(best.NodeID) == "" {
		return nil
	}

	path := make([]string, 0, 4)
	seen := make(map[string]struct{}, len(parentByNodeID)+1)
	currentNodeID := best.NodeID
	for strings.TrimSpace(currentNodeID) != "" {
		if _, ok := seen[currentNodeID]; ok {
			break
		}
		seen[currentNodeID] = struct{}{}
		path = append(path, currentNodeID)
		currentNodeID = strings.TrimSpace(parentByNodeID[currentNodeID])
	}

	steps := make([]domain.TreeTraversalStep, 0, len(path))
	for idx := len(path) - 1; idx >= 0; idx-- {
		nodeID := path[idx]
		hit, ok := scoreByNodeID[nodeID]
		if !ok {
			node := nodeByID[nodeID]
			hit = domain.TreeNodeHit{
				NodeID:    node.NodeID,
				Title:     node.Title,
				Level:     node.Level,
				StartPage: node.StartPage,
				EndPage:   node.EndPage,
			}
		}
		steps = append(steps, domain.TreeTraversalStep{
			NodeID:    hit.NodeID,
			Title:     hit.Title,
			Level:     hit.Level,
			Score:     hit.Score,
			StartPage: hit.StartPage,
			EndPage:   hit.EndPage,
		})
	}

	return steps
}

// bm25Scores pontua cada texto pela consulta com BM25 (Robertson & Zaragoza,
// k1=1,2, b=0,75), com acentos dobrados nos dois lados. Devolve o score de
// cada posicao, ja multiplicado por 100 e arredondado, para caber nos ints
// dos hits e manter os pesos relativos (no x3, pagina x2, tabela x5) que o
// scorer de substring usava.
//
// Por que BM25 e nao a contagem de substring: o experimento do artigo
// (internal/retrieval, bracos A e B) mediu P@1 de 0,56 contra 0,75 sobre o
// mesmo corpus. Na colecao da faculdade a contagem de substring fazia paginas
// longas do PPC vencerem a linha curta da matriz que tinha a resposta, porque
// a consulta carrega "em", "que", "de" e toda pagina longa contem todas. O
// IDF do BM25 e a normalizacao por comprimento resolvem exatamente isso.
// O braco A do experimento continua sendo a contagem de substring: ele mede o
// sistema como foi publicado; a producao passa a ser o braco B com dobra de
// acentos.
func bm25Scores(query string, texts []string) []int {
	docs := make([]retrieval.Document, len(texts))
	for i, text := range texts {
		docs[i] = retrieval.Document{ID: strconv.Itoa(i), Text: retrieval.FoldDiacritics(text)}
	}
	// A consulta passa pelo tokenizador de producao antes do BM25 para manter
	// a expansao numerica ("seven" tambem vira "7"), que o benchmark publico
	// exercita e o tokenizador do experimento nao faz.
	// A consulta ganha a variante singular/plural de cada termo: "estagio
	// extracurricular" tambem casa "estagios extracurriculares" (p. 199 do
	// PPC). So a consulta muda; o indice fica como esta, entao um termo a mais
	// nunca tira uma pagina, so soma quando a variante existe nela.
	queryText := retrieval.FoldDiacritics(strings.Join(expandNumberVariants(tokenizeRetrievalText(query)), " "))
	scores := make([]int, len(texts))
	for _, result := range retrieval.NewBM25Index(docs, retrieval.DefaultBM25Params()).Search(queryText) {
		i, err := strconv.Atoi(result.DocID)
		if err == nil && i >= 0 && i < len(scores) {
			scores[i] = int(math.Round(result.Score * 100))
		}
	}
	return scores
}

// significantQueryTokens sao os tokens da pergunta que contam como evidencia:
// acento e caixa dobrados, pontuacao das pontas fora ("operacionais?" e
// "operacionais"), sem conectivos nem numeros soltos.
func significantQueryTokens(query string) []string {
	tokens := connectionTokens(query)
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !ehPalavraVazia(token) {
			out = append(out, token)
		}
	}
	return out
}

// queryTermCoverage conta quantos dos termos (ja normalizados por
// significantQueryTokens) aparecem no texto como token inteiro.
func queryTermCoverage(text string, terms []string) int {
	tokens := tokenSet(text)
	n := 0
	for _, term := range terms {
		if _, ok := tokens[term]; ok {
			n++
		}
	}
	return n
}

// expandNumberVariants acrescenta a cada termo alfabetico de 4+ letras a sua
// forma no outro numero gramatical (regras de plural do portugues, so as
// regulares). Termos ja presentes nao se repetem.
func expandNumberVariants(tokens []string) []string {
	seen := make(map[string]bool, len(tokens)*2)
	out := make([]string, 0, len(tokens)*2)
	add := func(t string) {
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, token := range tokens {
		add(token)
		// So termos de 6+ letras: "curso"/"forma"/"eixo" ganhavam a variante
		// e puxavam paginas genericas (p24 do gabarito saiu das fontes);
		// "estagio", "extracurricular", "resolucao" sao os que precisam.
		if len(token) < 6 || isNumericToken(token) {
			continue
		}
		// Hifen conta como letra: "pre-requisitos" precisa da variante
		// "pre-requisito", que e o cabecalho da coluna da Matriz.
		for _, r := range token {
			if !unicode.IsLetter(r) && r != '-' {
				token = ""
				break
			}
		}
		if token == "" {
			continue
		}
		n := len(token)
		switch {
		case strings.HasSuffix(token, "oes") || strings.HasSuffix(token, "ões"):
			add(strings.TrimSuffix(strings.TrimSuffix(token, "oes"), "ões") + "ao")
		case strings.HasSuffix(token, "ais") || strings.HasSuffix(token, "eis") || strings.HasSuffix(token, "ois"):
			add(token[:n-2] + "l")
		case strings.HasSuffix(token, "ns"):
			add(token[:n-2] + "m")
		case strings.HasSuffix(token, "res") && n > 5:
			add(token[:n-2])
		case strings.HasSuffix(token, "s"):
			add(token[:n-1])
		case strings.HasSuffix(token, "ao") || strings.HasSuffix(token, "ão"):
			add(strings.TrimSuffix(strings.TrimSuffix(token, "ao"), "ão") + "oes")
		case strings.HasSuffix(token, "l"):
			add(token[:n-1] + "is")
		case strings.HasSuffix(token, "m"):
			add(token[:n-1] + "ns")
		case strings.HasSuffix(token, "r"):
			add(token + "es")
		default:
			add(token + "s")
		}
	}
	return out
}

// Bracos do experimento pre-registrado de enriquecimento
// (docs/prds/pre-registro-perguntas-geradas.md). Selecionados por variavel de
// ambiente para que os tres possam ser medidos sobre o MESMO indice, sem
// reindexar entre eles: reindexar entre bracos trocaria duas coisas ao mesmo
// tempo e a comparacao perderia o sentido.
//
// O padrao preserva o comportamento historico (conteudo + resumo), entao quem
// nao conhece a variavel nao muda de regime sem saber.
const (
	pageFieldsContent          = "content"
	pageFieldsContentSummary   = "content+summary"
	pageFieldsContentQuestions = "content+questions"
	// pageFieldsMaxQuestions pontua conteudo e perguntas SEPARADAMENTE e fica
	// com o maior dos dois: a pagina e relevante se o conteudo dela casa a
	// pergunta OU se uma das perguntas geradas casa.
	//
	// Maximo, e nao soma ponderada, de proposito: peso e um botao de ajuste, e
	// ajustar o peso ate o numero melhorar e a mesma armadilha de rodar varios
	// prompts e reportar o melhor. O maximo nao tem parametro para girar.
	pageFieldsMaxQuestions = "max(content,questions)"
)

// normalizePageSearchFields valida o braco configurado. Valor desconhecido cai
// no padrao e avisa: um erro de digitacao na variavel mediria silenciosamente o
// braco errado, que e o pior resultado possivel numa comparacao. Chamada UMA vez,
// na construcao de RetrievalOptions (antes era por pagina de cada pergunta, e o
// aviso inundava o log).
func normalizePageSearchFields(raw string) string {
	switch v := strings.TrimSpace(raw); v {
	case "":
		return pageFieldsContentSummary
	case pageFieldsContent, pageFieldsContentSummary, pageFieldsContentQuestions, pageFieldsMaxQuestions:
		return v
	default:
		log.Printf("THINGINDEX_PAGE_SEARCH_FIELDS=%q desconhecido; usando %q", v, pageFieldsContentSummary)
		return pageFieldsContentSummary
	}
}

// pageSearchText monta o texto da pagina que vai ao BM25, conforme o braco.
func pageSearchText(page models.DocumentPage, fields string) string {
	switch fields {
	case pageFieldsContent:
		return page.Content
	case pageFieldsContentQuestions:
		return strings.Join([]string{page.GeneratedQuestions, page.Content}, " ")
	default:
		return strings.Join([]string{page.Summary, page.Content}, " ")
	}
}

// buildCollectionRetrieval usa as opcoes padrao (content+summary). Mantida para
// os testes; o caminho de producao usa buildCollectionRetrievalWith com as
// opcoes lidas uma vez na subida do processo.
func buildCollectionRetrieval(
	query string,
	documents []models.Document,
	pages []models.DocumentPage,
	nodes []models.DocumentOutlineNode,
	chunks []models.DocumentChunk,
	tables []models.TableSection,
) *domain.CollectionRetrievalResult {
	return buildCollectionRetrievalWith(DefaultRetrievalOptions(), query, documents, pages, nodes, chunks, tables)
}

func buildCollectionRetrievalWith(
	opts RetrievalOptions,
	query string,
	documents []models.Document,
	pages []models.DocumentPage,
	nodes []models.DocumentOutlineNode,
	chunks []models.DocumentChunk,
	tables []models.TableSection,
) *domain.CollectionRetrievalResult {
	pageFields := opts.pageSearchFields()
	queryTerms := tokenizeRetrievalText(query)
	if len(queryTerms) == 0 {
		return nil
	}

	documentScores := make(map[uint]int, len(documents))
	documentTitles := make(map[uint]string, len(documents))
	// Termos significativos da pergunta (sem conectivos), por token inteiro:
	// e o que uma linha de tabela precisa casar para ser fonte, e o que o
	// titulo do documento precisa casar para reforcar as paginas dele.
	significant := significantQueryTokens(query)
	// Reforco por titulo foi testado e descartado (2026-09-09): "Sistemas de
	// Informacao" esta em quase toda pergunta e no titulo do PPC, e o reforco
	// puxava as tabelas do PPC na frente da linha certa da Matriz (perdeu 3
	// perguntas de tabela, ganhou nenhuma). O titulo so pontua o documento.
	for _, doc := range documents {
		documentTitles[doc.ID] = doc.Title
		if score := retrievalScore(doc.Title, queryTerms); score > 0 {
			documentScores[doc.ID] += score * 100
		}
	}

	nodeTexts := make([]string, len(nodes))
	for i, node := range nodes {
		nodeTexts[i] = outlineNodeSearchText(node)
	}
	nodeScores := bm25Scores(query, nodeTexts)
	// Ranking de documentos por MELHOR evidencia de cada tipo (no, pagina,
	// chunk, tabela), nao pela soma de todas: com a soma, um documento longo
	// com muitas paginas fracas vencia o documento com a tabela certa.
	best := make(map[uint]map[string]int, len(documents))
	keep := func(documentID uint, kind string, score int) {
		if best[documentID] == nil {
			best[documentID] = map[string]int{}
		}
		if score > best[documentID][kind] {
			best[documentID][kind] = score
		}
	}
	nodeHits := make([]domain.CollectionNodeHit, 0)
	for i, node := range nodes {
		score := nodeScores[i] * 3
		if score == 0 {
			continue
		}
		// Mesmos bonus do scorer anterior, na mesma escala (x100).
		score += metadataIntValue(node.Metadata, "confidence_score") / 25 * 100
		if node.Level > 1 {
			score += 100
		}

		keep(node.DocumentID, "node", score)
		nodeHits = append(nodeHits, domain.CollectionNodeHit{
			DocumentID: node.DocumentID,
			NodeID:     node.NodeID,
			Title:      node.Title,
			Summary:    node.Summary,
			Score:      score,
			StartPage:  node.StartPage,
			EndPage:    node.EndPage,
		})
	}

	sort.Slice(nodeHits, func(i, j int) bool {
		if nodeHits[i].Score == nodeHits[j].Score {
			if nodeHits[i].DocumentID == nodeHits[j].DocumentID {
				return nodeHits[i].NodeID < nodeHits[j].NodeID
			}
			return nodeHits[i].DocumentID < nodeHits[j].DocumentID
		}
		return nodeHits[i].Score > nodeHits[j].Score
	})
	if len(nodeHits) > 8 {
		nodeHits = nodeHits[:8]
	}

	pageTexts := make([]string, len(pages))
	for i, page := range pages {
		pageTexts[i] = pageSearchText(page, pageFields)
	}
	pageScores := bm25Scores(query, pageTexts)
	// Braco de maximo: o campo de perguntas e pontuado no seu PROPRIO corpus,
	// e nao colado ao conteudo. Colar dilui, porque o BM25 normaliza por
	// comprimento e 40 palavras de pergunta somem em 400 de conteudo; pontuar
	// a parte deixa o casamento curto e exato valer o que ele vale.
	if pageFields == pageFieldsMaxQuestions {
		perguntaTexts := make([]string, len(pages))
		for i, page := range pages {
			perguntaTexts[i] = page.GeneratedQuestions
		}
		for i, s := range bm25Scores(query, perguntaTexts) {
			if s > pageScores[i] {
				pageScores[i] = s
			}
		}
	}
	pageHits := make([]domain.CollectionPageHit, 0)
	for i, page := range pages {
		score := pageScores[i] * 2
		if score == 0 {
			continue
		}
		keep(page.DocumentID, "page", score)
		pageHits = append(pageHits, domain.CollectionPageHit{
			DocumentID: page.DocumentID,
			PageNumber: page.PageNumber,
			Snippet:    clampRetrievalSnippet(page.Content, 220),
			Score:      score,
		})
	}
	sort.Slice(pageHits, func(i, j int) bool {
		if pageHits[i].Score == pageHits[j].Score {
			if pageHits[i].DocumentID == pageHits[j].DocumentID {
				return pageHits[i].PageNumber < pageHits[j].PageNumber
			}
			return pageHits[i].DocumentID < pageHits[j].DocumentID
		}
		return pageHits[i].Score > pageHits[j].Score
	})
	if len(pageHits) > 8 {
		pageHits = pageHits[:8]
	}

	chunkTexts := make([]string, len(chunks))
	for i, chunk := range chunks {
		if chunk.ChunkType != "table_section" {
			chunkTexts[i] = strings.Join([]string{chunk.Title, chunk.Content}, " ")
		}
	}
	for i, score := range bm25Scores(query, chunkTexts) {
		if score > 0 {
			keep(chunks[i].DocumentID, "chunk", score*2)
		}
	}
	// Cada LINHA de tabela e uma unidade, com o cabecalho colado em cada
	// celula. Antes so as 3 primeiras linhas entravam, e a tabela so pontuava
	// o documento: "pre-requisito de X" caia numa pagina de prosa.
	type rowRef struct {
		table int
		row   int
	}
	rowRefs := make([]rowRef, 0)
	rowTexts := make([]string, 0)
	for t, table := range tables {
		for r, row := range table.Rows {
			rowRefs = append(rowRefs, rowRef{t, r})
			rowTexts = append(rowTexts, strings.TrimSpace(table.Title+" "+indexing.RenderTableRow(table.Headers, row)))
		}
	}
	tableRowHits := make([]domain.CollectionTableRowHit, 0)
	for i, score := range bm25Scores(query, rowTexts) {
		if score == 0 {
			continue
		}
		// Linha curta com UM termo da pergunta pontua alto no BM25 por ser
		// curta ("Descricao: Equivalencia de Estagio" para qualquer pergunta
		// com "estagio"); so linha que casa dois termos significativos e
		// fonte. Pergunta de um termo so exige esse um.
		if need := min(2, len(significant)); queryTermCoverage(rowTexts[i], significant) < need {
			continue
		}
		score *= 5
		table := tables[rowRefs[i].table]
		keep(table.DocumentID, "table", score)
		if table.PageNumber == nil || *table.PageNumber <= 0 {
			continue // sem pagina nao e fonte auditavel (HTML, planilha)
		}
		tableRowHits = append(tableRowHits, domain.CollectionTableRowHit{
			DocumentID: table.DocumentID,
			TableID:    table.ID,
			RowIndex:   rowRefs[i].row,
			PageNumber: *table.PageNumber,
			Title:      table.Title,
			Text:       indexing.RenderTableRow(table.Headers, table.Rows[rowRefs[i].row]),
			Score:      score,
		})
	}
	sort.SliceStable(tableRowHits, func(i, j int) bool { return tableRowHits[i].Score > tableRowHits[j].Score })
	if len(tableRowHits) > 8 {
		tableRowHits = tableRowHits[:8]
	}
	for documentID, kinds := range best {
		for _, score := range kinds {
			documentScores[documentID] += score
		}
	}

	documentHits := make([]domain.CollectionDocumentHit, 0, len(documentScores))
	for documentID, score := range documentScores {
		documentHits = append(documentHits, domain.CollectionDocumentHit{
			DocumentID: documentID,
			Title:      documentTitles[documentID],
			Score:      score,
		})
	}
	sort.Slice(documentHits, func(i, j int) bool {
		if documentHits[i].Score == documentHits[j].Score {
			return documentHits[i].DocumentID < documentHits[j].DocumentID
		}
		return documentHits[i].Score > documentHits[j].Score
	})
	if len(documentHits) > 5 {
		documentHits = documentHits[:5]
	}

	return &domain.CollectionRetrievalResult{
		Query:        query,
		DocumentHits: documentHits,
		NodeHits:     nodeHits,
		PageHits:     pageHits,
		TableRowHits: tableRowHits,
		Strategy:     "collection-bm25",
	}
}

func outlineNodeSearchText(node models.DocumentOutlineNode) string {
	retrievalText := metadataStringValue(node.Metadata, "retrieval_text")
	if retrievalText == "" {
		retrievalText = node.Summary
	}
	return strings.TrimSpace(strings.Join([]string{node.Title, retrievalText}, " "))
}

func tokenizeRetrievalText(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	// Letra maiuscula sozinha ("conceito D", "TCC I") e termo: sem ela a
	// pergunta sobre o conceito D achava qualquer pagina com "conceito". Letra
	// minuscula sozinha continua fora (artigo, preposicao).
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "\n", " ", "\t", " ", "?", " ", "!", " ")
	rawParts := strings.Fields(replacer.Replace(trimmed))
	parts := make([]string, 0, len(rawParts))
	standaloneUpper := make(map[string]bool, 2)
	for _, raw := range rawParts {
		if runes := []rune(raw); len(runes) == 1 && unicode.IsUpper(runes[0]) && !strings.ContainsRune("AOEÀÉ", runes[0]) {
			// "A matrícula" e artigo em inicio de frase, nao um "A" de conceito.
			standaloneUpper[strings.ToLower(raw)] = true
		}
		parts = append(parts, strings.ToLower(raw))
	}
	seen := make(map[string]struct{}, len(parts))
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, expanded := range expandNumericRetrievalTerms(part) {
			if len(expanded) < 1 {
				continue
			}
			if len(expanded) < 2 && !isNumericToken(expanded) && !standaloneUpper[expanded] {
				continue
			}
			if _, ok := seen[expanded]; ok {
				continue
			}
			seen[expanded] = struct{}{}
			tokens = append(tokens, expanded)
		}
	}
	return tokens
}

func expandNumericRetrievalTerms(term string) []string {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil
	}

	expansions := []string{term}
	switch term {
	case "zero":
		expansions = append(expansions, "0")
	case "one":
		expansions = append(expansions, "1")
	case "two":
		expansions = append(expansions, "2")
	case "three":
		expansions = append(expansions, "3")
	case "four":
		expansions = append(expansions, "4")
	case "five":
		expansions = append(expansions, "5")
	case "six":
		expansions = append(expansions, "6")
	case "seven":
		expansions = append(expansions, "7")
	case "eight":
		expansions = append(expansions, "8")
	case "nine":
		expansions = append(expansions, "9")
	case "ten":
		expansions = append(expansions, "10")
	case "0":
		expansions = append(expansions, "zero")
	case "1":
		expansions = append(expansions, "one")
	case "2":
		expansions = append(expansions, "two")
	case "3":
		expansions = append(expansions, "three")
	case "4":
		expansions = append(expansions, "four")
	case "5":
		expansions = append(expansions, "five")
	case "6":
		expansions = append(expansions, "six")
	case "7":
		expansions = append(expansions, "seven")
	case "8":
		expansions = append(expansions, "eight")
	case "9":
		expansions = append(expansions, "nine")
	case "10":
		expansions = append(expansions, "ten")
	}

	return expansions
}

func isNumericToken(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return value != ""
}

func retrievalScore(text string, queryTerms []string) int {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" || len(queryTerms) == 0 {
		return 0
	}

	score := 0
	for _, term := range queryTerms {
		if strings.Contains(normalized, term) {
			score++
		}
	}
	return score
}

func scoreCollectionPage(page models.DocumentPage, queryTerms []string, pageFields string) int {
	return retrievalScore(pageSearchText(page, pageFields), queryTerms) * 2
}

func scoreTreePage(page models.DocumentPage, queryTerms []string, pageFields string) int {
	return scoreCollectionPage(page, queryTerms, pageFields)
}

func scoreCollectionChunk(chunk models.DocumentChunk, queryTerms []string) int {
	if chunk.ChunkType == "table_section" {
		return 0
	}
	return retrievalScore(strings.Join([]string{chunk.Title, chunk.Content}, " "), queryTerms) * 2
}

func scoreTreeChunk(chunk models.DocumentChunk, queryTerms []string) int {
	score := scoreCollectionChunk(chunk, queryTerms)
	if score == 0 {
		return 0
	}
	if metadataIntValue(chunk.Metadata, "related_table_count") > 0 {
		score++
	}
	return score
}

func chunkPageRange(chunk models.DocumentChunk) (int, int, bool) {
	startPage := metadataIntValue(chunk.Metadata, "page_start")
	endPage := metadataIntValue(chunk.Metadata, "page_end")
	if startPage <= 0 || endPage < startPage {
		return 0, 0, false
	}
	return startPage, endPage, true
}

func renderTableRowsForRetrieval(rows [][]string, limit int) string {
	if len(rows) == 0 || limit <= 0 {
		return ""
	}

	parts := make([]string, 0, min(limit, len(rows)))
	for idx := 0; idx < len(rows) && idx < limit; idx++ {
		parts = append(parts, strings.Join(rows[idx], " "))
	}

	return strings.Join(parts, " ")
}

func clampRetrievalSnippet(value string, maxLen int) string {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len([]rune(cleaned)) <= maxLen {
		return cleaned
	}
	return strings.TrimSpace(string([]rune(cleaned)[:maxLen]))
}
