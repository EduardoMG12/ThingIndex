package evalcorpus

import (
	"sort"
	"strings"
)

// RetrievalUnit e uma unidade sobre a qual o experimento ranqueia.
//
// Existe para separar a UNIDADE DE RECUPERACAO do DOCUMENTO. Ate a issue #16 o
// experimento tratava as duas como a mesma coisa, e isso descasava da tese: o
// trabalho afirma proveniencia de paragrafo e media recuperacao de documento
// inteiro.
type RetrievalUnit struct {
	// ID e o identificador da unidade: o id da secao quando o documento tem
	// secoes declaradas, ou o id do proprio documento quando nao tem.
	ID string
	// DocumentID preserva a proveniencia. Toda unidade sabe de que documento
	// veio, o que permite reportar as duas granularidades lado a lado sem
	// reprocessar nada.
	DocumentID string
	// SectionID e vazio quando a unidade e o documento inteiro (degradacao do
	// RNF-1). Distinguir isso de "secao cujo id coincide com o do documento"
	// evita contar degradacao como granularidade fina na hora de reportar.
	SectionID string
	// Text e o conteudo da unidade -- o recorte da secao, nao o documento
	// completo. Se fosse o documento inteiro, todas as unidades de um mesmo
	// arquivo teriam score identico e o ranking em nivel de secao seria
	// indistinguivel do ranking em nivel de documento.
	Text string
	// Title e rotulo humano para as tabelas do artigo.
	Title string
}

// RetrievalUnits converte documentos nas unidades sobre as quais os bracos
// ranqueiam.
//
// Documento COM secoes declaradas vira uma unidade por secao, cada uma
// contendo o texto entre a propria ancora e a ancora seguinte. Documento SEM
// secoes vira uma unidade unica com o conteudo inteiro -- a degradacao exigida
// pelo RNF-1 da spec 001, que mantem o experimento executavel enquanto o corpus
// nao estiver totalmente anotado.
//
// A ordem de saida e deterministica (documento na ordem do corpus, secao por
// posicao no texto). Sem isso o ranking herdaria ordem de iteracao de mapa e o
// experimento deixaria de ser reproduzivel -- o mesmo defeito que este trabalho
// denuncia no indice vetorial aproximado.
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

// sliceSections recorta o conteudo de um documento nos limites das ancoras.
func sliceSections(doc Document) []RetrievalUnit {
	type marcador struct {
		section Section
		início  int
	}

	marcadores := make([]marcador, 0, len(doc.Sections))
	for _, section := range doc.Sections {
		início := localizarAncora(doc.Content, section)
		if início < 0 {
			// Validate ja recusa ancora inexistente na carga; aqui a defesa
			// existe para o caso de RetrievalUnits ser chamada com um corpus
			// montado em memoria sem passar por Load.
			continue
		}
		marcadores = append(marcadores, marcador{section: section, início: início})
	}

	// Ordena por posicao no texto: a ordem de declaracao no JSON nao precisa
	// coincidir com a ordem do documento, e o recorte depende da posicao.
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

// localizarAncora devolve a posicao inicial da secao no conteudo.
//
// CharStart desempata quando a mesma ancora aparece mais de uma vez: a busca
// comeca a partir dele. Zero significa "primeira ocorrencia", que e o caso
// comum e nao precisa ser anotado.
func localizarAncora(content string, section Section) int {
	if section.CharStart > 0 && section.CharStart <= len(content) {
		if offset := strings.Index(content[section.CharStart:], section.Anchor); offset >= 0 {
			return section.CharStart + offset
		}
	}
	return strings.Index(content, section.Anchor)
}
