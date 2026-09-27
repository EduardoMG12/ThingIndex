package evalcorpus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Gabarito e o conjunto de perguntas de coordenador sobre a colecao real de SI,
// usado pelo experimento de geracao (cmd/generation-eval).
//
// Difere do Corpus C1 de proposito: C1 e hermetico e mede RECUPERACAO sobre
// documentos sinteticos; o gabarito aponta para PDFs reais, por pagina fisica,
// e mede a RESPOSTA ponta a ponta. O hash congela as perguntas: ajustar o
// sistema e depois "ajustar" a pergunta e a forma mais facil de produzir um
// numero falso, e o teste falha se o arquivo mudar sem o hash acompanhar.
type Gabarito struct {
	Version      string             `json:"version"`
	Notes        string             `json:"notes"`
	FrozenSHA256 string             `json:"frozen_sha256"`
	ReviewedBy   []string           `json:"reviewed_by"`
	Documents    []GabaritoDocument `json:"documents"`
	Questions    []GabaritoQuestion `json:"questions"`
	// Strata e MinQuestions permitem que cada gabarito declare as proprias
	// regras. Sem isso, as regras do acervo de ensino (prosa, tabela,
	// calendario, negativa) valeriam para todo gabarito, e um corpus de
	// legislacao so entraria inventando perguntas de calendario sobre lei
	// federal ou afrouxando a constante para todo mundo. Ausentes, valem os
	// padroes historicos, entao o gabarito de ensino nao muda.
	Strata       map[string]int `json:"strata,omitempty"`
	MinQuestions int            `json:"min_questions,omitempty"`
}

// strata devolve os estratos exigidos deste gabarito.
func (g *Gabarito) strata() map[string]int {
	if len(g.Strata) > 0 {
		return g.Strata
	}
	return GabaritoStrata
}

// minQuestions devolve o minimo de perguntas deste gabarito.
func (g *Gabarito) minQuestions() int {
	if g.MinQuestions > 0 {
		return g.MinQuestions
	}
	return GabaritoMinQuestions
}

type GabaritoDocument struct {
	ID    string `json:"id"`
	File  string `json:"file"`
	Title string `json:"title"`
	Pages int    `json:"pages"`
	// Aliases sao os outros nomes com que o documento aparece nas fontes do
	// chat: o titulo que ficou gravado na colecao nem sempre e o do gabarito
	// ("NR 6 - Equipamentos de Protecao Individual" contra "NR 6 - EPI"). Sem
	// eles a citacao correta nao casa com o alvo e a proveniencia sai
	// subestimada. Ficam fora do hash congelado, que cobre so as perguntas:
	// dizer como um documento se chama nao muda o que se pergunta nem o que
	// se espera.
	Aliases []string `json:"aliases,omitempty"`
}

// DocumentResolver devolve o id do gabarito a partir do nome com que o
// documento aparece numa fonte (id, titulo ou alias), com caixa e acento
// dobrados. Nome desconhecido volta como veio.
func (g *Gabarito) DocumentResolver(fold func(string) string) func(string) string {
	porNome := map[string]string{}
	for _, d := range g.Documents {
		for _, nome := range append([]string{d.ID, d.Title}, d.Aliases...) {
			if n := fold(strings.TrimSpace(nome)); n != "" {
				porNome[n] = d.ID
			}
		}
	}
	return func(nome string) string {
		if id, ok := porNome[fold(strings.TrimSpace(nome))]; ok {
			return id
		}
		return nome
	}
}

type GabaritoTarget struct {
	DocumentID string `json:"document_id"`
	Pages      []int  `json:"pages"`
}

type GabaritoQuestion struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Stratum string `json:"stratum"`
	// Targets lista documento e paginas onde a resposta esta. Vazio marca
	// pergunta negativa: a resposta certa e a recusa.
	Targets []GabaritoTarget `json:"targets"`
	// AnswerPatterns sao regex que TODAS precisam casar na resposta, ja com
	// acentos e caixa dobrados. Uma alternancia dentro da regex cobre as formas
	// equivalentes ("sis07206|programacao de computadores i").
	AnswerPatterns []string `json:"answer_patterns"`
	// RejectPatterns invalida a resposta se QUALQUER uma casar, mesmo que as
	// AnswerPatterns casem. Serve as perguntas cuja resposta certa pode conter
	// "nao" ou "ha": sem isso, uma recusa ("nao ha informacoes sobre isso")
	// passava como acerto de existencia ou de premissa falsa. A banca mediu
	// que uma recusa constante acertava as 24 perguntas dessas formas.
	RejectPatterns []string `json:"reject_patterns,omitempty"`
	Expected       string   `json:"expected"`
	// Variant e PairID sustentam o achado central do trabalho sobre reuso de
	// vocabulario: a recuperacao acerta quase tudo quando a pergunta repete as
	// palavras do documento e despenca quando a mesma necessidade e expressa de
	// outro jeito.
	//
	// Medir isso exige que a MESMA necessidade de informacao apareca duas vezes,
	// com a mesma resposta esperada e o mesmo alvo, mudando so a formulacao:
	//
	//	literal   -- usa os termos do documento
	//	parafrase -- expressa a necessidade com o vocabulario de quem pergunta
	//
	// PairID liga as duas. Sem o par, um gabarito escrito por quem ja leu a
	// pagina herda o vocabulario dela e mede casamento lexical se passando por
	// compreensao, que foi exatamente o diagnostico do conselho sobre os 100%
	// obtidos nos primeiros temas.
	Variant string `json:"variant,omitempty"`
	PairID  string `json:"pair_id,omitempty"`
	// Origin: "pdf" (escrita a partir do PDF) ou "rodada-cc5" (ja usada para
	// ajustar o sistema; o artigo reporta com e sem elas).
	Origin string `json:"origin"`
}

// GabaritoStrata sao os estratos PADRAO, do acervo de ensino, e o minimo de
// perguntas em cada um. Um gabarito pode declarar os seus em "strata".
var GabaritoStrata = map[string]int{"prosa": 20, "tabela": 15, "calendario": 10, "negativa": 5}

// GabaritoMinQuestions e o minimo PADRAO. Um gabarito pode declarar o seu em
// "min_questions".
const GabaritoMinQuestions = 50

// Variantes de formulacao. Ver o comentario de GabaritoQuestion.Variant.
const (
	VariantLiteral   = "literal"
	VariantParafrase = "parafrase"
)

func LoadGabarito(path string) (*Gabarito, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ler gabarito: %w", err)
	}
	var g Gabarito
	if err := json.Unmarshal(raw, &g); err != nil {
		return nil, fmt.Errorf("decodificar gabarito: %w", err)
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &g, nil
}

// QuestionsSHA256 e o hash das perguntas serializadas; e o valor esperado em
// FrozenSHA256.
func (g *Gabarito) QuestionsSHA256() string {
	raw, _ := json.Marshal(g.Questions)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (g *Gabarito) Validate() error {
	estratos := g.strata()
	if minimo := g.minQuestions(); len(g.Questions) < minimo {
		return fmt.Errorf("gabarito tem %d perguntas; minimo %d", len(g.Questions), minimo)
	}
	if got := g.QuestionsSHA256(); got != g.FrozenSHA256 {
		return fmt.Errorf("frozen_sha256 nao confere: arquivo %q, perguntas %q (recalcule e registre em Discoveries)", g.FrozenSHA256, got)
	}
	pages := map[string]int{}
	for _, d := range g.Documents {
		if d.ID == "" || d.File == "" || d.Pages <= 0 {
			return fmt.Errorf("documento %q invalido", d.ID)
		}
		pages[d.ID] = d.Pages
	}
	seen := map[string]bool{}
	perStratum := map[string]int{}
	pares := map[string][]GabaritoQuestion{}
	for _, q := range g.Questions {
		if seen[q.ID] {
			return fmt.Errorf("pergunta %q duplicada", q.ID)
		}
		seen[q.ID] = true
		if _, ok := estratos[q.Stratum]; !ok {
			aceitos := make([]string, 0, len(estratos))
			for s := range estratos {
				aceitos = append(aceitos, s)
			}
			sort.Strings(aceitos)
			return fmt.Errorf("pergunta %q: estrato %q desconhecido; aceitos neste gabarito: %v", q.ID, q.Stratum, aceitos)
		}
		perStratum[q.Stratum]++
		if q.Origin != "pdf" && q.Origin != "rodada-cc5" {
			return fmt.Errorf("pergunta %q: origin %q desconhecida", q.ID, q.Origin)
		}
		if q.Text == "" || len(q.AnswerPatterns) == 0 {
			return fmt.Errorf("pergunta %q sem texto ou sem padrao de resposta", q.ID)
		}
		for _, p := range q.AnswerPatterns {
			if _, err := regexp.Compile(p); err != nil {
				return fmt.Errorf("pergunta %q: regex %q: %w", q.ID, p, err)
			}
		}
		switch q.Variant {
		case "", VariantLiteral, VariantParafrase:
		default:
			return fmt.Errorf("pergunta %q: variant %q desconhecida (use %q ou %q)", q.ID, q.Variant, VariantLiteral, VariantParafrase)
		}
		if (q.Variant == "") != (q.PairID == "") {
			return fmt.Errorf("pergunta %q: variant e pair_id andam juntos, veio variant=%q pair_id=%q", q.ID, q.Variant, q.PairID)
		}
		if q.PairID != "" {
			pares[q.PairID] = append(pares[q.PairID], q)
		}
		if (q.Stratum == "negativa") != (len(q.Targets) == 0) {
			return fmt.Errorf("pergunta %q: negativa sem alvo e vice-versa", q.ID)
		}
		for _, t := range q.Targets {
			total, ok := pages[t.DocumentID]
			if !ok {
				return fmt.Errorf("pergunta %q: documento %q nao declarado", q.ID, t.DocumentID)
			}
			if len(t.Pages) == 0 {
				return fmt.Errorf("pergunta %q: alvo %q sem paginas", q.ID, t.DocumentID)
			}
			for _, p := range t.Pages {
				if p < 1 || p > total {
					return fmt.Errorf("pergunta %q: pagina %d fora de 1..%d em %q", q.ID, p, total, t.DocumentID)
				}
			}
		}
	}
	names := make([]string, 0, len(estratos))
	for s := range estratos {
		names = append(names, s)
	}
	sort.Strings(names)
	for _, s := range names {
		if perStratum[s] < estratos[s] {
			return fmt.Errorf("estrato %q tem %d perguntas; minimo %d", s, perStratum[s], estratos[s])
		}
	}

	// Um par incompleto ou incoerente invalida a comparacao literal x parafrase,
	// que e justamente o que o par existe para medir.
	pairIDs := make([]string, 0, len(pares))
	for id := range pares {
		pairIDs = append(pairIDs, id)
	}
	sort.Strings(pairIDs)
	for _, id := range pairIDs {
		grupo := pares[id]
		if len(grupo) != 2 {
			return fmt.Errorf("par %q tem %d perguntas; precisa de exatamente 2 (literal e parafrase)", id, len(grupo))
		}
		if grupo[0].Variant == grupo[1].Variant {
			return fmt.Errorf("par %q tem duas perguntas %q; precisa de uma de cada variante", id, grupo[0].Variant)
		}
		if grupo[0].Expected != grupo[1].Expected {
			return fmt.Errorf("par %q: as duas variantes precisam esperar a MESMA resposta, senao nao sao a mesma necessidade de informacao", id)
		}
		if grupo[0].Stratum != grupo[1].Stratum {
			return fmt.Errorf("par %q: as duas variantes precisam ter a mesma forma de pergunta", id)
		}
	}
	return nil
}

// IsNegative diz se a resposta certa e a recusa.
func (q GabaritoQuestion) IsNegative() bool { return len(q.Targets) == 0 }

// TargetPages devolve as paginas-alvo por documento.
func (q GabaritoQuestion) TargetPages() map[string][]int {
	out := map[string][]int{}
	for _, t := range q.Targets {
		out[t.DocumentID] = append(out[t.DocumentID], t.Pages...)
	}
	return out
}
