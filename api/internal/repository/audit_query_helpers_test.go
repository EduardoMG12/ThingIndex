package repository

import (
	"strings"
	"testing"

	"thing-index/internal/domain"
)

func TestLimiteDeAuditoriaEhNormalizado(t *testing.T) {
	// limit vem do cliente: 0 daria pagina vazia e 100000 varreria a tabela.
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
	// Sem escape, "%" casaria com todos os registros.
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
	obtido := escapeLikePattern("  CONTRATO Social  ")

	if obtido != "contrato social" {
		t.Fatalf("esperava %q, veio %q", "contrato social", obtido)
	}
}

func TestEscapeDePadraoPreservaTextoComum(t *testing.T) {
	obtido := escapeLikePattern("Contrato de prestacao")

	if obtido != "contrato de prestacao" {
		t.Fatalf("texto comum foi alterado: %q", obtido)
	}
	if strings.Contains(obtido, `\`) {
		t.Fatalf("escape aplicado a texto sem coringa: %q", obtido)
	}
}

func TestEscapeDePadraoLidaComEntradaVazia(t *testing.T) {
	if obtido := escapeLikePattern(""); obtido != "" {
		t.Fatalf("entrada vazia virou %q", obtido)
	}
	if obtido := escapeLikePattern("     "); obtido != "" {
		t.Fatalf("apenas espacos virou %q", obtido)
	}
}

func TestEscapeDePadraoNeutralizaBarraInvertida(t *testing.T) {
	// A barra e o caractere de escape. Compara com o valor exato: strings.Contains
	// passava com o codigo bugado porque a entrada ja tinha a sequencia.
	obtido := escapeLikePattern(`caminho\arquivo`)
	esperado := `caminho\\arquivo`

	if obtido != esperado {
		t.Fatalf("escapeLikePattern(%q) = %q, esperado %q -- barra crua desarma o escape dos demais coringas",
			`caminho\arquivo`, obtido, esperado)
	}
}

func TestFiltroComQueryNulaNaoEntraEmPanico(t *testing.T) {
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
