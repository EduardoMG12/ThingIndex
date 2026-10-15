// Package evalcorpus carrega o corpus de avaliacao da recuperacao.
//
// O gabarito e por ID de documento/secao (nunca por titulo extraido pelo
// pipeline), graduado 0-3 e inclui casos negativos.
package evalcorpus

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Corpus struct {
	Version   string     `json:"version"`
	Notes     string     `json:"notes,omitempty"`
	Documents []Document `json:"documents"`
	Queries   []Query    `json:"queries"`
}

type Document struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	// Stratum: tipo de documento (juridico, tabular, escaneado...).
	Stratum string `json:"stratum,omitempty"`
	// Sections sao as unidades intra-documento ranqueaveis. Vazio: o
	// documento entra como unidade unica.
	Sections []Section `json:"sections,omitempty"`
}

// Section e uma unidade recuperavel dentro de um documento. O ID e estavel e
// escrito a mao, nunca derivado do extrator; Anchor e o texto do documento que
// marca o inicio da secao e precisa existir no conteudo.
type Section struct {
	ID     string `json:"id"`
	Anchor string `json:"anchor"`
	// CharStart desempata ancoras repetidas; zero = primeira ocorrencia.
	CharStart int `json:"char_start,omitempty"`
	// Title e so rotulo de leitura; nao entra no casamento.
	Title string `json:"title,omitempty"`
}

type Query struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Variant: "literal" (copia termos do alvo), "paraphrase" ou "noisy"
	// (erro de digitacao/termo impreciso).
	Variant string `json:"variant"`
	// Relevance: document_id -> grau 0-3.
	Relevance map[string]int `json:"relevance"`
	// RelevanceSections: section_id -> grau 0-3 (gabarito em nivel de no).
	RelevanceSections map[string]int `json:"relevance_sections,omitempty"`
	// Negative e explicito para nao confundir "ninguem anotou" com "nada e
	// relevante".
	Negative bool `json:"negative,omitempty"`
}

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

// Validate recusa corpus malformado antes de qualquer medicao: erro de
// gabarito nao falha, so produz um numero plausivel e errado.
func (c *Corpus) Validate() error {
	if len(c.Documents) == 0 {
		return fmt.Errorf("corpus sem documentos")
	}
	if len(c.Queries) == 0 {
		return fmt.Errorf("corpus sem consultas")
	}

	seenDocs := map[string]bool{}
	// seenSections e global porque a consulta referencia a secao so pelo id.
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
			// Ancora com erro de digitacao nunca casaria e a culpa cairia no braco.
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

		for docID, grade := range query.Relevance {
			if !seenDocs[docID] {
				return fmt.Errorf("consulta %q julga o documento inexistente %q", query.ID, docID)
			}
			if grade < 0 || grade > 3 {
				return fmt.Errorf("consulta %q tem grau %d para %q; a escala e 0-3", query.ID, grade, docID)
			}
		}

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

		// Documento relevante com secoes declaradas exige julgar qual secao
		// responde; senao o gabarito de no fica incompleto em silencio.
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

// CheckCircularity avisa sobre consultas paraphrase/noisy cujos termos estao
// todos no documento-alvo (grau 3): na pratica sao literais.
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

// CheckSectionAnchoring avisa quando o id da secao e a propria ancora (fora
// pontuacao): o gabarito passaria a depender do titulo que o extrator produz.
func (c *Corpus) CheckSectionAnchoring() []string {
	var warnings []string

	for _, doc := range c.Documents {
		for _, section := range doc.Sections {
			// So o sufixo apos '#': "doc-a#s3" -> "s3".
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

// normalizarRotulo: "3 DISPOSITIVO", "3. Dispositivo" e "3-dispositivo" viram "3dispositivo".
func normalizarRotulo(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if ('a' <= r && r <= 'z') || ('0' <= r && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

type Stats struct {
	Documents      int
	Queries        int
	ByVariant      map[string]int
	NegativeCases  int
	JudgedPairs    int
	AvgQueryTokens float64
}

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
