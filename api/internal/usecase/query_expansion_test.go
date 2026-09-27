package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

// O prompt esta congelado pelo pre-registro
// (docs/prds/pre-registro-expansao-consulta.md). Este teste existe para que
// alterar o texto quebre o build: rodar varias versoes do prompt e reportar a
// melhor produz um numero favoravel e sem valor, e ninguem detecta isso depois.
func TestPromptDaExpansaoEstaCongelado(t *testing.T) {
	const esperado = "bbc051efd4d6636b2682c7ffd33866845aaf894da6ad0172ccbe289bf44d2250"
	sum := sha256.Sum256([]byte(queryExpansionSystemPrompt))
	got := hex.EncodeToString(sum[:])
	if got != esperado {
		t.Fatalf("prompt da expansao mudou: hash %s.\n"+
			"Se a mudanca e intencional, ela exige um pre-registro NOVO e a medicao recomeca.", got)
	}

	// Conteudo que o pre-registro promete, para o hash nao virar um numero solto.
	for _, exigido := range []string{
		"Do not answer the question",
		"technical or formal terms",
		"At most 8 terms",
		"return an empty line",
	} {
		if !strings.Contains(queryExpansionSystemPrompt, exigido) {
			t.Errorf("prompt perdeu a instrucao %q", exigido)
		}
	}
}

func TestSanitizeExpansion(t *testing.T) {
	casos := []struct {
		nome  string
		bruto string
		quer  string
	}{
		{"linha simples", "EPI equipamento protecao individual", "epi equipamento protecao individual"},
		{"corta na primeira linha", "epi fornecimento\nExplicacao: o empregador...", "epi fornecimento"},
		{"tira pontuacao e duplicata", "EPI, epi; fornecimento.", "epi fornecimento"},
		{"descarta termo de uma letra", "a o epi", "epi"},
		{"vazio vira vazio", "   ", ""},
		{"aspas ao redor", "\"epi fornecimento\"", "epi fornecimento"},
		{"no maximo oito termos", "a1 b2 c3 d4 e5 f6 g7 h8 i9 j10", "a1 b2 c3 d4 e5 f6 g7 h8"},
		// O modelo as vezes ignora a instrucao e responde a pergunta. Colar uma
		// resposta inteira na consulta arruinaria o ranking, entao o texto longo
		// e descartado por inteiro em vez de truncado.
		{"resposta longa e descartada", strings.Repeat("palavra ", 40), ""},
	}
	for _, c := range casos {
		if got := sanitizeExpansion(c.bruto); got != c.quer {
			t.Errorf("%s: sanitizeExpansion(%q) = %q, queria %q", c.nome, c.bruto, got, c.quer)
		}
	}
}

func TestChaveDaExpansaoNormaliza(t *testing.T) {
	a := queryExpansionKey("Quem paga o capacete?")
	b := queryExpansionKey("  quem   PAGA o capacete?  ")
	if a != b {
		t.Fatalf("caixa e espaco deviam dar a mesma chave: %s vs %s", a, b)
	}
	if a == queryExpansionKey("Quem paga a luva?") {
		t.Fatal("perguntas diferentes nao podem colidir")
	}
}

type llmFalso struct {
	resposta string
	err      error
	chamadas int
}

func (l *llmFalso) GenerateText(context.Context, string, string) (string, error) {
	l.chamadas++
	return l.resposta, l.err
}

type cacheFalso struct {
	entradas map[string]domain.EnrichmentCacheEntry
	gravadas int
}

func chaveCacheFalso(scope domain.EnrichmentCacheScope, kind, hash string) string {
	return fmt.Sprintf("%d|%s|%s|%s|%s", scope.UserID, scope.Provider, scope.Model, kind, hash)
}

func (c *cacheFalso) LookupEnrichment(_ context.Context, scope domain.EnrichmentCacheScope, kind string, hashes []string) (map[string]domain.EnrichmentCacheEntry, error) {
	out := map[string]domain.EnrichmentCacheEntry{}
	for _, h := range hashes {
		if e, ok := c.entradas[chaveCacheFalso(scope, kind, h)]; ok {
			out[h] = e
		}
	}
	return out, nil
}

func (c *cacheFalso) StoreEnrichment(_ context.Context, scope domain.EnrichmentCacheScope, entries []domain.EnrichmentCacheEntry) error {
	for _, e := range entries {
		c.entradas[chaveCacheFalso(scope, e.Kind, e.ContentHash)] = e
		c.gravadas++
	}
	return nil
}

var escopoM1 = domain.EnrichmentCacheScope{UserID: 1, Provider: "OPENAI|api.openai.com", Model: "m1"}

// Regressao P0-1: a expansao da pergunta usava o mesmo cache global do
// enriquecimento. Termos gerados pelo provedor de um usuario nao podem ser
// servidos a outro.
func TestExpandNaoReaproveitaExpansaoDeOutroUsuario(t *testing.T) {
	cache := &cacheFalso{entradas: map[string]domain.EnrichmentCacheEntry{}}
	atacante := &llmFalso{resposta: "termo envenenado"}
	vitima := &llmFalso{resposta: "epi equipamento"}
	e := newQueryExpander(cache)

	escopoAtacante := domain.EnrichmentCacheScope{UserID: 1, Provider: "OPENAI|evil.example.com", Model: "gpt-4o-mini"}
	escopoVitima := domain.EnrichmentCacheScope{UserID: 2, Provider: "OPENAI|api.openai.com", Model: "gpt-4o-mini"}
	e.Expand(context.Background(), atacante, escopoAtacante, "Quem paga o capacete?")

	if got := e.Expand(context.Background(), vitima, escopoVitima, "Quem paga o capacete?"); got != "epi equipamento" {
		t.Fatalf("a vitima recebeu a expansao de outro usuario: %q", got)
	}
	if vitima.chamadas != 1 {
		t.Fatalf("a vitima devia chamar o proprio LLM; chamadas: %d", vitima.chamadas)
	}
}

func TestExpandUsaCacheNaSegundaVez(t *testing.T) {
	cache := &cacheFalso{entradas: map[string]domain.EnrichmentCacheEntry{}}
	llm := &llmFalso{resposta: "epi equipamento"}
	e := newQueryExpander(cache)

	primeira := e.Expand(context.Background(), llm, escopoM1, "Quem paga o capacete?")
	segunda := e.Expand(context.Background(), llm, escopoM1, "quem   paga o CAPACETE?")

	if primeira != "epi equipamento" || segunda != primeira {
		t.Fatalf("expansoes divergiram: %q e %q", primeira, segunda)
	}
	if llm.chamadas != 1 {
		t.Fatalf("a segunda chamada devia vir do cache; chamadas de LLM: %d", llm.chamadas)
	}
}

// Expansao vazia tambem vai para o cache: sem isso, toda pergunta que ja usa o
// vocabulario formal pagaria uma chamada de LLM em cada repeticao da bateria, e
// a medicao de custo do experimento sairia errada.
func TestExpansaoVaziaTambemEGravada(t *testing.T) {
	cache := &cacheFalso{entradas: map[string]domain.EnrichmentCacheEntry{}}
	llm := &llmFalso{resposta: "\n"}
	e := newQueryExpander(cache)

	e.Expand(context.Background(), llm, escopoM1, "Qual o prazo decadencial?")
	e.Expand(context.Background(), llm, escopoM1, "Qual o prazo decadencial?")

	if llm.chamadas != 1 {
		t.Fatalf("expansao vazia devia ficar em cache; chamadas: %d", llm.chamadas)
	}
	if cache.gravadas != 1 {
		t.Fatalf("esperava uma gravacao, houve %d", cache.gravadas)
	}
}

// Falha de LLM cai no braco de controle em vez de derrubar a pergunta: a busca
// sem expansao e o comportamento atual do sistema, nao um erro.
func TestExpandFalhaSilenciosaNaoQuebraABusca(t *testing.T) {
	e := newQueryExpander(&cacheFalso{entradas: map[string]domain.EnrichmentCacheEntry{}})
	llm := &llmFalso{err: errors.New("provedor fora do ar")}
	if got := e.Expand(context.Background(), llm, escopoM1, "Quem paga o capacete?"); got != "" {
		t.Fatalf("esperava expansao vazia, veio %q", got)
	}
	if got := e.Expand(context.Background(), nil, escopoM1, "Quem paga o capacete?"); got != "" {
		t.Fatalf("sem LLM a expansao tem que ser vazia, veio %q", got)
	}
}

func TestBracoDesligadoPorPadrao(t *testing.T) {
	t.Setenv("THINGINDEX_QUERY_EXPANSION", "")
	if RetrievalOptionsFromEnv().QueryExpansion {
		t.Fatal("o braco de controle e o padrao: sem a variavel, a expansao fica desligada")
	}
	t.Setenv("THINGINDEX_QUERY_EXPANSION", "1")
	if !RetrievalOptionsFromEnv().QueryExpansion {
		t.Fatal("THINGINDEX_QUERY_EXPANSION=1 devia ligar a expansao")
	}
}
