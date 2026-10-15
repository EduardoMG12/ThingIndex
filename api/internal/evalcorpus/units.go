package evalcorpus

import (
	"sort"
	"strings"
)

// RetrievalUnit e a unidade ranqueada: uma secao ou o documento inteiro.
type RetrievalUnit struct {
	// ID: id da secao, ou do documento quando ele nao tem secoes.
	ID         string
	DocumentID string
	// SectionID vazio = unidade e o documento inteiro.
	SectionID string
	// Text e so o recorte da secao, senao todas as secoes do arquivo empatariam.
	Text  string
	Title string
}

// RetrievalUnits gera uma unidade por secao (texto da ancora ate a proxima) ou,
// sem secoes, uma unidade com o documento inteiro. A ordem e deterministica:
// documentos na ordem do corpus, secoes por posicao no texto.
func RetrievalUnits(docs []Document) []RetrievalUnit {
	var units []RetrievalUnit

	for _, doc := range docs {
		if len(doc.Sections) == 0 {
			units = append(units, RetrievalUnit{
				ID:         doc.ID,
				DocumentID: doc.ID,
				Text:       strings.TrimSpace(doc.Title + "\n" + doc.Content),
				Title:      doc.Title,
			})
			continue
		}

		units = append(units, sliceSections(doc)...)
	}

	return units
}

func sliceSections(doc Document) []RetrievalUnit {
	type marcador struct {
		section Section
		início  int
	}

	marcadores := make([]marcador, 0, len(doc.Sections))
	for _, section := range doc.Sections {
		início := localizarAncora(doc.Content, section)
		if início < 0 {
			// Corpus montado em memoria nao passou por Validate.
			continue
		}
		marcadores = append(marcadores, marcador{section: section, início: início})
	}

	// A ordem no JSON nao precisa seguir a do documento.
	sort.SliceStable(marcadores, func(i, j int) bool {
		return marcadores[i].início < marcadores[j].início
	})

	units := make([]RetrievalUnit, 0, len(marcadores))
	for i, m := range marcadores {
		fim := len(doc.Content)
		if i+1 < len(marcadores) {
			fim = marcadores[i+1].início
		}

		units = append(units, RetrievalUnit{
			ID:         m.section.ID,
			DocumentID: doc.ID,
			SectionID:  m.section.ID,
			Text:       strings.TrimSpace(doc.Content[m.início:fim]),
			Title:      m.section.Title,
		})
	}

	return units
}

// localizarAncora busca a ancora a partir de CharStart (desempate de ancoras
// repetidas).
func localizarAncora(content string, section Section) int {
	if section.CharStart > 0 && section.CharStart <= len(content) {
		if offset := strings.Index(content[section.CharStart:], section.Anchor); offset >= 0 {
			return section.CharStart + offset
		}
	}
	return strings.Index(content, section.Anchor)
}
