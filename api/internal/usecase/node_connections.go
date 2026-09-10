package usecase

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"thing-index/internal/domain"
	"thing-index/internal/models"
	"thing-index/internal/retrieval"
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
	originText := outlineNodeSearchText(origin)
	terms := connectionTerms(originText)
	originCitations := normCitations(originText)
	if len(terms) == 0 && len(originCitations) == 0 {
		return nil
	}

	type candidateIndex struct {
		node      models.DocumentOutlineNode
		tokens    map[string]struct{}
		citations map[string]struct{}
	}
	// Conexao util liga documentos DIFERENTES. Trechos do mesmo arquivo
	// compartilham vocabulario por construcao e afogariam o sinal real.
	indexed := make([]candidateIndex, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.DocumentID == origin.DocumentID {
			continue
		}
		text := outlineNodeSearchText(candidate)
		indexed = append(indexed, candidateIndex{node: candidate, tokens: tokenSet(text), citations: toSet(normCitations(text))})
	}

	// So termo RARO no acervo sustenta uma aresta. "curso", "edital", "campus"
	// nao sao conectivos, mas aparecem em centenas de secoes da mesma colecao
	// e ligavam anexos sem relacao nenhuma. A raridade e medida aqui mesmo,
	// sobre os candidatos, por TOKEN INTEIRO: por substring "lei" casava
	// "leitura" e "estagio" casava "estagiario".
	limit := int(math.Ceil(float64(len(indexed)) * NodeConnectionMaxTermShare))
	if limit < 2 {
		limit = 2
	}
	rare := make([]string, 0, len(terms))
	for _, term := range terms {
		df := 0
		for _, c := range indexed {
			if _, ok := c.tokens[term]; ok {
				df++
				if df > limit {
					break
				}
			}
		}
		if df <= limit {
			rare = append(rare, term)
		}
	}

	// Score zero nunca vira conexao, mesmo com minScore <= 0: aceitar zero
	// devolveria todo par do acervo como relacionado.
	threshold := minScore
	if threshold < 1 {
		threshold = 1
	}

	connections := make([]domain.NodeConnection, 0, len(indexed))
	for _, c := range indexed {
		matched := make([]string, 0, len(rare))
		for _, term := range rare {
			if _, ok := c.tokens[term]; ok {
				matched = append(matched, term)
			}
		}
		sameNorm := make([]string, 0, len(originCitations))
		for _, citation := range originCitations {
			if _, ok := c.citations[citation]; ok {
				sameNorm = append(sameNorm, citation)
			}
		}
		// Citar a mesma norma e relacao por si so ("Lei 9.394/1996" nos dois),
		// mesmo sem termo raro em comum; vale por dois termos.
		score := len(matched) + 2*len(sameNorm)
		if score < threshold || (len(matched) == 0 && len(sameNorm) == 0) {
			continue
		}
		kind := NodeConnectionKindRareTerm
		if len(sameNorm) > 0 {
			kind = NodeConnectionKindSameNorm
		}
		sort.Strings(matched)
		connections = append(connections, domain.NodeConnection{
			DocumentID:   c.node.DocumentID,
			NodeID:       c.node.NodeID,
			Title:        c.node.Title,
			Summary:      c.node.Summary,
			StartPage:    c.node.StartPage,
			Score:        score,
			Kind:         kind,
			MatchedTerms: append(sameNorm, matched...),
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
	// Um no com 200 arestas nao esta "muito conectado": e um hub de vocabulario,
	// e no canvas vira um novelo. Ficam as mais fortes.
	if len(connections) > NodeConnectionMaxDegree {
		connections = connections[:NodeConnectionMaxDegree]
	}

	return connections
}

const (
	NodeConnectionKindRareTerm = "termo-raro"
	NodeConnectionKindSameNorm = "mesma-norma"
	// NodeConnectionMaxDegree e o teto de arestas por no.
	NodeConnectionMaxDegree = 20
)

// NodeConnectionMaxTermShare e a fracao maxima de secoes do acervo em que um
// termo pode aparecer e ainda contar como evidencia de conexao. Com 1.076
// secoes na colecao de SI, 2% sao 22 secoes: acima disso e vocabulario do
// acervo ("objetivos", "conceitos"), nao relacao. Piso de 2 para conjuntos
// pequenos. Era 5% quando o casamento era por substring; por token o df
// cai e o teto precisou apertar junto.
const NodeConnectionMaxTermShare = 0.02

// tokenSet dobra acento e caixa e devolve os tokens do texto como conjunto.
func tokenSet(text string) map[string]struct{} {
	tokens := connectionTokens(text)
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

// connectionTokens dobra acento e caixa e tira pontuacao das pontas: "a)" e
// "(b" sao marcadores de lista, nao termos, e por token inteiro eles casavam
// entre quaisquer duas listas.
func connectionTokens(text string) []string {
	raw := tokenizeRetrievalText(retrieval.FoldDiacritics(text))
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		token = strings.TrimFunc(token, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}

// connectionTerms devolve os termos da origem que podem sustentar uma aresta:
// tokens inteiros, sem conectivos, sem numeros e sem nome proprio.
func connectionTerms(text string) []string {
	properNouns := titleCaseOnlyWords(text)
	tokens := connectionTokens(text)
	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if ehPalavraVazia(token) {
			continue
		}
		if _, isName := properNouns[token]; isName {
			continue
		}
		terms = append(terms, token)
	}
	return terms
}

var wordEdgePunct = strings.NewReplacer(",", "", ".", "", ";", "", ":", "", "(", "", ")", "", "\"", "", "'", "")

// titleCaseOnlyWords devolve as palavras que no texto so aparecem com inicial
// maiuscula e resto minusculo ("Andreia", "Marini", "Paulo"): nome proprio,
// que liga secoes por quem assinou, nao pelo assunto. Palavra em CAIXA ALTA
// (titulo) ou em minusculas em algum lugar continua valendo. A primeira
// palavra do texto nao conta: comeca maiuscula por ser inicio de frase.
func titleCaseOnlyWords(text string) map[string]struct{} {
	titleCase := map[string]bool{}
	otherCase := map[string]bool{}
	for i, raw := range strings.Fields(text) {
		word := wordEdgePunct.Replace(raw)
		runes := []rune(word)
		if len(runes) < 2 {
			continue
		}
		key := strings.ToLower(retrieval.FoldDiacritics(word))
		rest := string(runes[1:])
		switch {
		case unicode.IsUpper(runes[0]) && strings.ToLower(rest) == rest && i > 0:
			titleCase[key] = true
		default:
			otherCase[key] = true
		}
	}
	names := make(map[string]struct{})
	for key := range titleCase {
		if !otherCase[key] {
			names[key] = struct{}{}
		}
	}
	return names
}

// normCitationRE reconhece "Lei nº 9.394/1996", "Resolução CONSUP 82/2022",
// "Decreto 7.824/2012", "Portaria 18/2012", "Parecer CNE/CES 136/2012".
var normCitationRE = regexp.MustCompile(`(?i)\b(lei|resolu[cç][aã]o|decreto|portaria|parecer|instru[cç][aã]o normativa)\b[^\d\n]{0,24}?(n[º°o.]?\s*)?(\d[\d.]*)(?:/(\d{2,4}))?`)

// normCitations devolve as citacoes de norma do texto normalizadas para
// "tipo numero/ano" (numero sem pontos), em ordem estavel e sem repeticao.
func normCitations(text string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, m := range normCitationRE.FindAllStringSubmatch(text, -1) {
		kind := strings.ToLower(retrieval.FoldDiacritics(m[1]))
		kind = strings.Join(strings.Fields(kind), " ")
		number := strings.ReplaceAll(m[3], ".", "")
		if number == "" || len(number) > 8 {
			continue
		}
		citation := kind + " " + number
		if m[4] != "" {
			citation += "/" + m[4]
		}
		if _, ok := seen[citation]; ok {
			continue
		}
		seen[citation] = struct{}{}
		out = append(out, citation)
	}
	sort.Strings(out)
	return out
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
	// Aparato de referencia bibliografica: liga qualquer duas ementas.
	"isbn": {}, "ed": {}, "edicao": {}, "editora": {}, "org": {}, "vol": {}, "capitulo": {}, "cap": {},
	// Conectivos e genericos que a amostra do demo mostrou justificando aresta.
	"acoes": {}, "atender": {}, "conjunto": {}, "realizacao": {}, "gerais": {}, "geral": {},
	"desta": {}, "deste": {}, "destas": {}, "destes": {}, "dessa": {}, "desse": {}, "dessas": {}, "desses": {}, "nesta": {}, "neste": {},
	// Interrogativos e verbos de pergunta: estao em toda pergunta e em
	// nenhuma resposta; contavam como "termo casado" da fonte.
	"qual": {}, "quais": {}, "quando": {}, "quanto": {}, "quanta": {}, "quantos": {}, "quantas": {}, "onde": {}, "quem": {},
	"porque": {}, "existe": {}, "existem": {}, "tem": {}, "ha": {}, "diz": {}, "significa": {}, "esta": {}, "estao": {},
}

// ehPalavraVazia diz se o termo e conectivo ou numeracao de secao.
//
// Numeros isolados ("1", "2") entram porque sao numeracao de titulo --
// "1 DO OBJETO" e "1 DA DISPONIBILIDADE" casariam pelo "1" sem qualquer
// relacao de conteudo. Numeros com mais de um digito ficam de fora: "99,5",
// "30" e "2025" costumam ser valores contratuais com significado real.
func ehPalavraVazia(termo string) bool {
	if len(termo) == 1 || numericTokenRE.MatchString(termo) || romanNumeralRE.MatchString(termo) {
		return true
	}
	_, vazia := palavrasVaziasPT[termo]
	return vazia
}

// romanNumeralRE: "vi", "xii" sao numeracao de capitulo e titulo, nao assunto.
var romanNumeralRE = regexp.MustCompile(`^(i|ii|iii|iv|v|vi|vii|viii|ix|x|xi|xii|xiii|xiv|xv|xvi|xvii|xviii|xix|xx)$`)

// numericTokenRE: anos, numeros de resolucao, valores. Sozinhos nao dizem de
// que assunto uma secao trata; como citacao de norma entram por normCitations.
var numericTokenRE = regexp.MustCompile(`^[\d.,/º°-]+$`)
