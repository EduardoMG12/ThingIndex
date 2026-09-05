package repository

import (
	"strings"
	"testing"

	"thing-index/internal/domain"
)

// internal/repository tinha 0,8% de cobertura porque quase todo teste do pacote
// exige Postgres e roda so com a tag integration. O resultado e que a suite
// padrao atravessa a camada que fala com o banco sem exercitar nada -- e o bug
// da corrida de colecao morava aqui.
//
// Estes testes cobrem a logica PURA: normalizacao de parametro e escape de
// padrao. Onde o banco e indispensavel, o teste continua com a tag integration;
// a meta nao e fingir cobertura com mock de SQL.

func TestLimiteDeAuditoriaEhNormalizado(t *testing.T) {
	// O limite vem da querystring, controlada pelo cliente. Sem normalizacao,
	// limit=0 devolveria pagina vazia (o usuario acha que nao ha eventos) e
	// limit=100000 varreria a tabela inteira -- negacao de servico com uma
	// requisicao.
	casos := []struct {
		nome     string
		entrada  int
		esperado int
		porque   string
	}{
		{"zero vira o padrao", 0, 20, "pagina vazia pareceria ausencia de eventos"},
		{"negativo vira o padrao", -5, 20, "valor negativo quebraria o LIMIT do SQL"},
		{"acima do teto e cortado", 100000, 100, "varreria a tabela inteira"},
		{"exatamente no teto passa", 100, 100, ""},
		{"valor comum passa intacto", 25, 25, ""},
		{"um passa intacto", 1, 1, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			obtido := clampAuditLimit(caso.entrada)
			if obtido != caso.esperado {
				t.Fatalf("clampAuditLimit(%d) = %d, esperado %d. %s",
					caso.entrada, obtido, caso.esperado, caso.porque)
			}
		})
	}
}

func TestEscapeDePadraoNeutralizaCoringas(t *testing.T) {
	// O termo de busca vai para um LIKE. Sem escape, um usuario digitando "%"
	// casaria com TODOS os registros -- e num filtro de auditoria isso significa
	// ver o que nao deveria.
	casos := []struct {
		nome     string
		entrada  string
		proibido string
		porque   string
	}{
		{
			nome:     "porcentagem e escapada",
			entrada:  "100%",
			proibido: `100%`,
			porque:   "% solto casa com qualquer sequencia",
		},
		{
			nome:     "sublinhado e escapado",
			entrada:  "nota_fiscal",
			proibido: `nota_fiscal`,
			porque:   "_ solto casa com qualquer caractere",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			obtido := escapeLikePattern(caso.entrada)
			if obtido == caso.proibido {
				t.Fatalf("escapeLikePattern(%q) devolveu %q sem escapar: %s",
					caso.entrada, obtido, caso.porque)
			}
			if !strings.Contains(obtido, `\`) {
				t.Fatalf("escapeLikePattern(%q) = %q: nenhum caractere foi escapado",
					caso.entrada, obtido)
			}
		})
	}
}

func TestEscapeDePadraoNormalizaCaixaEEspaco(t *testing.T) {
	// A busca e case-insensitive por decisao de produto, e espaco nas pontas
	// vem de copiar e colar. Sem normalizar, "  Contrato " nao acharia
	// "contrato".
	obtido := escapeLikePattern("  CONTRATO Social  ")

	if obtido != "contrato social" {
		t.Fatalf("esperava %q, veio %q", "contrato social", obtido)
	}
}

func TestEscapeDePadraoPreservaTextoComum(t *testing.T) {
	// Escape agressivo demais quebraria a busca legitima: o usuario que procura
	// "contrato" precisa achar "contrato".
	obtido := escapeLikePattern("Contrato de prestacao")

	if obtido != "contrato de prestacao" {
		t.Fatalf("texto comum foi alterado: %q", obtido)
	}
	if strings.Contains(obtido, `\`) {
		t.Fatalf("escape aplicado a texto sem coringa: %q", obtido)
	}
}

func TestEscapeDePadraoLidaComEntradaVazia(t *testing.T) {
	// Campo de busca em branco e o estado inicial da tela.
	if obtido := escapeLikePattern(""); obtido != "" {
		t.Fatalf("entrada vazia virou %q", obtido)
	}
	if obtido := escapeLikePattern("     "); obtido != "" {
		t.Fatalf("apenas espacos virou %q", obtido)
	}
}

func TestEscapeDePadraoNeutralizaBarraInvertida(t *testing.T) {
	// A barra e o proprio caractere de escape: deixa-la passar permitiria
	// desarmar o escape dos outros coringas.
	// A asserção compara com o esperado exato, e nao com strings.Contains: a
	// primeira versao deste teste passava com o codigo bugado porque a entrada
	// JA continha a sequencia procurada.
	obtido := escapeLikePattern(`caminho\arquivo`)
	esperado := `caminho\\arquivo`

	if obtido != esperado {
		t.Fatalf("escapeLikePattern(%q) = %q, esperado %q -- barra crua desarma o escape dos demais coringas",
			`caminho\arquivo`, obtido, esperado)
	}
}

func TestFiltroComQueryNulaNaoEntraEmPanico(t *testing.T) {
	// Guarda defensiva: query nula significa erro de montagem, e panico aqui
	// derrubaria o handler inteiro em vez de devolver erro.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panico com query nula: %v", r)
		}
	}()

	query, err := applyCommonAuditFilters(nil, domain.AuditPageFilter{}, false)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if query != nil {
		t.Fatal("query nula deveria continuar nula")
	}
}
