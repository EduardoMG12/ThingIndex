// Package evalcorpus carrega o corpus de avaliacao do experimento comparativo.
//
// O formato e deliberadamente diferente do corpus do benchmark atual
// (internal/usecase/testdata/indexing-benchmark/corpus.json), por tres razoes
// que o roadmap identificou como fatais para a validade do trabalho:
//
//  1. CIRCULARIDADE. No corpus antigo, todo `expected_top_node_title` e copia
//     literal de um item de `expected_outline_titles` do mesmo documento: o
//     gabarito de recuperacao e definido em funcao da saida estrutural do
//     proprio pipeline. Se o extrator renomear "3 DISPOSITIVO" para
//     "3. Dispositivo", a metrica quebra -- nao porque a recuperacao piorou,
//     mas porque a ROTULACAO mudou. Aqui o gabarito e por ID de documento,
//     nunca por titulo produzido pelo sistema.
//
//  2. RELEVANCIA BINARIA. O corpus antigo so sabe dizer "e o esperado" ou "nao
//     e". Sem graus nao existe nDCG, e sem nDCG nao da para distinguir "trouxe
//     o melhor documento em 1o" de "trouxe um documento tangencial em 1o".
//
//  3. AUSENCIA DE CASOS NEGATIVOS. O corpus antigo nao tem nenhuma consulta
//     cuja resposta correta seja "nenhum documento e relevante", entao nunca
//     mediu falso positivo. Um sistema que sempre responde alguma coisa pontua
//     igual a um que sabe dizer "nao encontrei".
package evalcorpus

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Corpus e o conjunto completo de avaliacao.
type Corpus struct {
	Version   string     `json:"version"`
	Notes     string     `json:"notes,omitempty"`
	Documents []Document `json:"documents"`
	Queries   []Query    `json:"queries"`
}

// Document e uma unidade recuperavel.
type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	// Stratum permite reportar resultado por tipo de documento (juridico,
	// tabular, escaneado...). Sem estrato, uma media global esconde que a
	// abordagem vai bem num tipo e mal em outro.
	Stratum string `json:"stratum,omitempty"`
	// Sections declara as unidades INTRA-documento sobre as quais o
	// experimento pode ranquear.
	//
	// Existe porque ranquear documento inteiro descasa da tese: o trabalho
	// afirma proveniencia de paragrafo e media recuperacao de documento. Com 5
	// documentos e 1 relevante por consulta, R@5 era 1,000 por aritmetica e a
	// baseline aleatoria de P@1 era 0,200 -- o experimento nao discriminava.
	//
	// Vazio e valido: o documento entra como unidade unica (degradacao
	// prevista no RNF-1 da spec 001).
	Sections []Section `json:"sections,omitempty"`
}

// Section e uma unidade recuperavel dentro de um documento.
//
// O ID e AUTORAL e ESTAVEL, jamais derivado da saida do extrator. Essa e a
// licao mais cara ja aprendida neste corpus: o gabarito antigo usava
// `expected_top_node_title`, copia literal do titulo que o proprio pipeline
// extraia, entao renomear a heuristica de heading derrubava a metrica sem que a
// recuperacao tivesse mudado. O corpus media autoconsistencia.
//
// Anchor e o texto do DOCUMENTO que marca o inicio da secao. Serve para casar a
// extracao com o gabarito sem que o gabarito dependa do extrator: se a ancora
// nao existir no texto-fonte, a carga falha (Validate) em vez de produzir um
// numero plausivel e errado.
type Section struct {
	ID     string `json:"id"`
	Anchor string `json:"anchor"`
	// CharStart e um desempate deterministico para quando a mesma ancora
	// aparece mais de uma vez no documento. Zero significa "primeira
	// ocorrencia".
	CharStart int `json:"char_start,omitempty"`
	// Title e rotulo humano para as tabelas do artigo. Nao participa de
	// nenhuma decisao de casamento -- e texto para leitura.
	Title string `json:"title,omitempty"`
}

// Query e uma necessidade de informacao com gabarito graduado.
type Query struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Variant registra como a consulta foi escrita: "literal" (copia termos do
	// alvo), "paraphrase" (mesma necessidade, outras palavras) ou "noisy"
	// (com erro de digitacao/termo impreciso).
	//
	// A QUEDA de P@1 de literal -> paraphrase -> noisy e o grafico mais
	// informativo do experimento: mede exatamente o que o benchmark atual
	// esconde ao escrever consultas copiando palavras do trecho-alvo.
	Variant string `json:"variant"`
	// Relevance mapeia document_id -> grau 0-3. Um mapa VAZIO marca um caso
	// negativo: a resposta correta e nao devolver nada.
	Relevance map[string]int `json:"relevance"`
	// RelevanceSections mapeia section_id -> grau 0-3, o gabarito em nivel de
	// no. Coexiste com Relevance de proposito: os dois niveis sao reportados
	// lado a lado no artigo, e manter ambos permite comparar o mesmo braco nas
	// duas granularidades.
	RelevanceSections map[string]int `json:"relevance_sections,omitempty"`
	// Negative torna o caso negativo explicito, em vez de inferido pela
	// ausencia de julgamentos. Distinguir "sem gabarito porque ninguem anotou"
	// de "sem gabarito porque nada e relevante" evita contar erro de anotacao
	// como acerto do sistema.
	Negative bool `json:"negative,omitempty"`
}

// Load le e valida o corpus.
func Load(path string) (*Corpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ler corpus: %w", err)
	}

	var corpus Corpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		return nil, fmt.Errorf("decodificar corpus: %w", err)
	}

	if err := corpus.Validate(); err != nil {
		return nil, err
	}

	return &corpus, nil
}

// Validate recusa corpus malformado ANTES de qualquer medicao.
//
// A validacao existe porque um erro de gabarito nao se manifesta como falha:
// ele se manifesta como um numero plausivel e errado, que so seria descoberto
// na arguicao. Barrar na carga e a unica forma de garantir que os numeros
// publicados vieram de um gabarito consistente.
func (c *Corpus) Validate() error {
	if len(c.Documents) == 0 {
		return fmt.Errorf("corpus sem documentos")
	}
	if len(c.Queries) == 0 {
		return fmt.Errorf("corpus sem consultas")
	}

	seenDocs := map[string]bool{}
	// secoesPorDoc guarda os ids validos por documento; seenSections e o
	// universo global, porque o gabarito de consulta referencia a secao pelo
	// id sem repetir o documento.
	secoesPorDoc := map[string][]Section{}
	seenSections := map[string]bool{}
	for _, doc := range c.Documents {
		if strings.TrimSpace(doc.ID) == "" {
			return fmt.Errorf("documento sem id")
		}
		if seenDocs[doc.ID] {
			return fmt.Errorf("documento duplicado: %q", doc.ID)
		}
		if strings.TrimSpace(doc.Content) == "" {
			return fmt.Errorf("documento %q sem conteudo", doc.ID)
		}
		seenDocs[doc.ID] = true

		for _, section := range doc.Sections {
			if strings.TrimSpace(section.ID) == "" {
				return fmt.Errorf("documento %q tem secao sem id", doc.ID)
			}
			if seenSections[section.ID] {
				return fmt.Errorf("secao duplicada: %q", section.ID)
			}
			seenSections[section.ID] = true

			if strings.TrimSpace(section.Anchor) == "" {
				return fmt.Errorf("secao %q sem ancora", section.ID)
			}
			// A ancora precisa existir no texto-fonte. Uma ancora com erro de
			// digitacao nao falharia em lugar nenhum: ela simplesmente nunca
			// casaria com a extracao, e o braco levaria a culpa por um erro de
			// anotacao.
			if !strings.Contains(doc.Content, section.Anchor) {
				return fmt.Errorf(
					"secao %q tem ancora %q que nao existe no conteudo do documento %q",
					section.ID, section.Anchor, doc.ID,
				)
			}
			if section.CharStart < 0 || section.CharStart > len(doc.Content) {
				return fmt.Errorf(
					"secao %q tem char_start %d fora do documento %q (0..%d)",
					section.ID, section.CharStart, doc.ID, len(doc.Content),
				)
			}
		}
		secoesPorDoc[doc.ID] = doc.Sections
	}

	seenQueries := map[string]bool{}
	for _, query := range c.Queries {
		if strings.TrimSpace(query.ID) == "" {
			return fmt.Errorf("consulta sem id")
		}
		if seenQueries[query.ID] {
			return fmt.Errorf("consulta duplicada: %q", query.ID)
		}
		seenQueries[query.ID] = true

		if strings.TrimSpace(query.Text) == "" {
			return fmt.Errorf("consulta %q sem texto", query.ID)
		}

		switch query.Variant {
		case "literal", "paraphrase", "noisy":
		default:
			return fmt.Errorf("consulta %q tem variante invalida %q (use literal, paraphrase ou noisy)", query.ID, query.Variant)
		}

		// Gabarito apontando para documento inexistente e o erro mais comum na
		// anotacao manual, e o mais traicoeiro: a consulta simplesmente nunca
		// e satisfeita, e o braco leva a culpa por um erro de digitacao.
		for docID, grade := range query.Relevance {
			if !seenDocs[docID] {
				return fmt.Errorf("consulta %q julga o documento inexistente %q", query.ID, docID)
			}
			if grade < 0 || grade > 3 {
				return fmt.Errorf("consulta %q tem grau %d para %q; a escala e 0-3", query.ID, grade, docID)
			}
		}

		// Mesma checagem em nivel de no.
		for sectionID, grade := range query.RelevanceSections {
			if !seenSections[sectionID] {
				return fmt.Errorf("consulta %q julga a secao inexistente %q", query.ID, sectionID)
			}
			if grade < 0 || grade > 3 {
				return fmt.Errorf("consulta %q tem grau %d para a secao %q; a escala e 0-3", query.ID, grade, sectionID)
			}
		}

		if query.Negative && len(query.Relevance) > 0 {
			return fmt.Errorf("consulta %q e negativa mas tem julgamentos de relevancia", query.ID)
		}
		if query.Negative && len(query.RelevanceSections) > 0 {
			return fmt.Errorf("consulta %q e negativa mas tem julgamentos de secao", query.ID)
		}
		if !query.Negative && len(query.Relevance) == 0 {
			return fmt.Errorf("consulta %q nao tem julgamento nenhum; marque negative:true se a ausencia for proposital", query.ID)
		}

		// Coerencia entre os dois niveis: se o documento julgado relevante tem
		// secoes declaradas, a consulta precisa dizer QUAL delas responde.
		// Sem isso o gabarito de no ficaria incompleto em silencio, e a
		// metrica cairia sem que a recuperacao tivesse piorado -- o mesmo modo
		// de falha que a ancora invalida provoca.
		for docID := range query.Relevance {
			if len(secoesPorDoc[docID]) == 0 {
				continue
			}
			temSecaoDoDoc := false
			for sectionID := range query.RelevanceSections {
				for _, section := range secoesPorDoc[docID] {
					if section.ID == sectionID {
						temSecaoDoDoc = true
						break
					}
				}
				if temSecaoDoDoc {
					break
				}
			}
			if !temSecaoDoDoc {
				return fmt.Errorf(
					"consulta %q julga o documento %q, que tem secoes declaradas, mas nao julga nenhuma secao dele",
					query.ID, docID,
				)
			}
		}
	}

	return nil
}

// CheckCircularity procura o defeito que invalidou o benchmark anterior:
// consultas cujo texto e copia literal de um trecho do documento-alvo.
//
// Quando isso acontece, o teste vira "um matcher de substring encontra o
// paragrafo cujas palavras eu copiei" -- tautologico, e um grep acerta 100%.
//
// A checagem e por sobreposicao de tokens: se TODOS os termos da consulta
// aparecem no documento-alvo, ela e literal por construcao. Isso e aceitavel
// para consultas marcadas como "literal" (que existem justamente para medir o
// teto), mas e um defeito silencioso em paraphrase e noisy.
func (c *Corpus) CheckCircularity(tokenize func(string) []string) []string {
	byID := map[string]Document{}
	for _, doc := range c.Documents {
		byID[doc.ID] = doc
	}

	var warnings []string
	for _, query := range c.Queries {
		if query.Variant == "literal" || query.Negative {
			continue
		}

		queryTokens := tokenize(query.Text)
		if len(queryTokens) == 0 {
			continue
		}

		for docID, grade := range query.Relevance {
			if grade < 3 {
				continue
			}

			docTokens := map[string]bool{}
			for _, token := range tokenize(byID[docID].Content) {
				docTokens[token] = true
			}

			contidos := 0
			for _, token := range queryTokens {
				if docTokens[token] {
					contidos++
				}
			}

			if contidos == len(queryTokens) {
				warnings = append(warnings, fmt.Sprintf(
					"consulta %q (variante %s) tem todos os termos presentes no documento %q: e literal na pratica, e mede o casamento em vez da recuperacao",
					query.ID, query.Variant, docID,
				))
			}
		}
	}

	sort.Strings(warnings)
	return warnings
}

// CheckSectionAnchoring procura o defeito que invalidou o benchmark anterior,
// agora na sua forma mais provavel de reaparecer: um id de secao derivado do
// titulo que o EXTRATOR produz.
//
// O corpus antigo usava `expected_top_node_title`, copia literal da saida
// estrutural do proprio pipeline. Renomear a heuristica de heading de
// "3 DISPOSITIVO" para "3. Dispositivo" derrubava a metrica -- nao porque a
// recuperacao piorou, mas porque a ROTULACAO mudou. O benchmark media
// autoconsistencia.
//
// Ao introduzir gabarito em nivel de no o risco volta: e tentador nomear a
// secao com o titulo dela. A regra e simples e mecanica: o id precisa ser um
// identificador ESTAVEL, nao uma frase do documento. Se o id (ignorando o
// prefixo do documento e a pontuacao) coincidir com a ancora, o gabarito
// voltou a depender do extrator.
func (c *Corpus) CheckSectionAnchoring() []string {
	var warnings []string

	for _, doc := range c.Documents {
		for _, section := range doc.Sections {
			// Considera so o sufixo apos '#': "doc-a#s3" -> "s3". Um id sem
			// '#' e avaliado inteiro, o que tambem pega "1 RELATORIO".
			sufixo := section.ID
			if idx := strings.LastIndex(section.ID, "#"); idx >= 0 {
				sufixo = section.ID[idx+1:]
			}

			if normalizarRotulo(sufixo) == normalizarRotulo(section.Anchor) {
				warnings = append(warnings, fmt.Sprintf(
					"secao %q do documento %q usa a propria ancora como identificador: o gabarito passa a depender do titulo que o extrator produz, exatamente o defeito que invalidou o benchmark anterior. Use um id estavel (ex.: %q)",
					section.ID, doc.ID, doc.ID+"#s1",
				))
			}
		}
	}

	sort.Strings(warnings)
	return warnings
}

// normalizarRotulo reduz um rotulo a letras e digitos minusculos, para que a
// comparacao nao dependa de pontuacao, acento de largura ou espacamento --
// "3 DISPOSITIVO", "3. Dispositivo" e "3-dispositivo" colapsam no mesmo valor.
func normalizarRotulo(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Stats resume o corpus para reporte no artigo.
type Stats struct {
	Documents      int
	Queries        int
	ByVariant      map[string]int
	NegativeCases  int
	JudgedPairs    int
	AvgQueryTokens float64
}

// Describe produz as estatisticas que o artigo precisa declarar. Um corpus
// pequeno demais ou sem casos negativos precisa ser visivel na tabela, e nao
// descoberto pela banca.
func (c *Corpus) Describe(tokenize func(string) []string) Stats {
	stats := Stats{
		Documents: len(c.Documents),
		Queries:   len(c.Queries),
		ByVariant: map[string]int{},
	}

	totalTokens := 0
	for _, query := range c.Queries {
		stats.ByVariant[query.Variant]++
		if query.Negative {
			stats.NegativeCases++
		}
		stats.JudgedPairs += len(query.Relevance)
		totalTokens += len(tokenize(query.Text))
	}

	if len(c.Queries) > 0 {
		stats.AvgQueryTokens = float64(totalTokens) / float64(len(c.Queries))
	}

	return stats
}
