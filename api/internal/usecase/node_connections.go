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

// BuildNodeConnections deriva as conexoes de um no com nos de outros
// documentos. As arestas sao recalculadas a cada chamada, nada e persistido.
// minScore evita que todo no ganhe conexao com o "menos ruim".
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
	// So documentos diferentes: trechos do mesmo arquivo compartilham vocabulario.
	indexed := make([]candidateIndex, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.DocumentID == origin.DocumentID {
			continue
		}
		text := outlineNodeSearchText(candidate)
		indexed = append(indexed, candidateIndex{node: candidate, tokens: tokenSet(text), citations: toSet(normCitations(text))})
	}

	// So termo raro no acervo sustenta aresta. Raridade por token inteiro: por
	// substring "lei" casava "leitura".
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

	// Score zero nunca vira conexao, mesmo com minScore <= 0.
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
		// Citar a mesma norma ja e relacao; vale por dois termos.
		score := len(matched) + 2*len(sameNorm)
		if score < threshold || (len(matched) == 0 && len(sameNorm) == 0) {
			continue
		}
		kind := NodeConnectionKindRareTerm
		if len(sameNorm) > 0 {
			kind = NodeConnectionKindSameNorm
			if strings.HasPrefix(sameNorm[0], "disciplina ") {
				kind = NodeConnectionKindSameDiscipline
			}
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

	// Score desc, NodeID asc para a ordem nao depender da entrada.
	sort.SliceStable(connections, func(i, j int) bool {
		if connections[i].Score != connections[j].Score {
			return connections[i].Score > connections[j].Score
		}
		return connections[i].NodeID < connections[j].NodeID
	})
	// No com centenas de arestas e hub de vocabulario; ficam as mais fortes.
	if len(connections) > NodeConnectionMaxDegree {
		connections = connections[:NodeConnectionMaxDegree]
	}

	return connections
}

const (
	NodeConnectionKindRareTerm       = "termo-raro"
	NodeConnectionKindSameNorm       = "mesma-norma"
	NodeConnectionKindSameDiscipline = "mesma-disciplina"
	// NodeConnectionMaxDegree e o teto de arestas por no.
	NodeConnectionMaxDegree = 20
)

// NodeConnectionMaxTermShare: fracao maxima de secoes em que um termo pode
// aparecer e ainda contar como evidencia. Piso de 2 para acervos pequenos.
const NodeConnectionMaxTermShare = 0.02

func tokenSet(text string) map[string]struct{} {
	tokens := connectionTokens(text)
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

// connectionTokens tira pontuacao das pontas: "a)" e "(b" casavam entre
// quaisquer duas listas.
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

// titleCaseOnlyWords devolve palavras que so aparecem capitalizadas (nome
// proprio). A primeira palavra do texto nao conta.
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

var normCitationRE = regexp.MustCompile(`(?i)\b(lei|resolu[cç][aã]o|decreto|portaria|parecer|instru[cç][aã]o normativa)\b[^\d\n]{0,24}?(n[º°o.]?\s*)?(\d[\d.]*)(?:/(\d{2,4}))?`)

// disciplineCodeRE: codigos como "SIS07302". Mesmo codigo e relacao, como
// citar a mesma norma.
var disciplineCodeRE = regexp.MustCompile(`\b[A-Z]{2,4}\d{4,6}\b`)

// normCitations normaliza para "tipo numero/ano", sem repeticao, mais os
// codigos de disciplina.
func normCitations(text string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, code := range disciplineCodeRE.FindAllString(text, -1) {
		citation := "disciplina " + code
		if _, ok := seen[citation]; ok {
			continue
		}
		seen[citation] = struct{}{}
		out = append(out, citation)
	}
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

// palavrasVaziasPT ligam qualquer par de documentos sem indicar relacao.
// Lista curta de proposito, para nao descartar termo com carga no dominio.
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

// ehPalavraVazia: numero de um digito e numeracao de titulo; numeros maiores
// costumam ser valores com significado.
func ehPalavraVazia(termo string) bool {
	if len(termo) == 1 || numericTokenRE.MatchString(termo) || romanNumeralRE.MatchString(termo) {
		return true
	}
	_, vazia := palavrasVaziasPT[termo]
	return vazia
}

// romanNumeralRE: "vi", "xii" sao numeracao de capitulo e titulo, nao assunto.
var romanNumeralRE = regexp.MustCompile(`^(i|ii|iii|iv|v|vi|vii|viii|ix|x|xi|xii|xiii|xiv|xv|xvi|xvii|xviii|xix|xx)$`)

// numericTokenRE: anos, numeros de resolucao, valores. Citacao de norma entra
// por normCitations.
var numericTokenRE = regexp.MustCompile(`^[\d.,/º°-]+$`)
