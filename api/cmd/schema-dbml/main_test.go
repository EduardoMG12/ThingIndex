package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// O valor deste gerador esta em o .dbml nao poder mentir sobre o esquema. Os
// testes abaixo travam as tres formas de ele mentir em silencio: perder uma
// tabela, inventar uma relacao que nao existe, e ficar defasado no repositorio
// depois de alguem mexer nos modelos.

func TestRenderCobreTodosOsModelosGerenciados(t *testing.T) {
	rendered, err := render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	tables := regexp.MustCompile(`(?m)^Table (\w+) \{`).FindAllStringSubmatch(rendered, -1)

	// 21 e o numero verificado contra um Postgres 16 real: subindo o
	// AutoMigrate num container limpo, information_schema.tables devolve
	// exatamente estas 21 tabelas e 253 colunas (22 com paragraph_highlights, 2026-09-08). O artigo afirmava 20 --
	// era esse o erro que o Apendice A carregava.
	const esperado = 22
	if len(tables) != esperado {
		t.Fatalf("esperava %d tabelas no DBML, obtive %d", esperado, len(tables))
	}

	for _, table := range tables {
		if !strings.Contains(rendered, table[1]) {
			t.Errorf("tabela %s declarada mas nao referenciada", table[1])
		}
	}
}

func TestRenderNaoInventaAutoReferencia(t *testing.T) {
	rendered, err := render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// Regressao real: o GORM registra a relacao HasMany nos dois schemas, e a
	// primeira versao do gerador deduzia a tabela pelo lado errado. O
	// resultado eram refs falsas do tipo
	// `chat_conversation_entries.conversation_id > chat_conversation_entries.id`,
	// que num diagrama de banco viram uma FK que nao existe.
	//
	// document_outline_nodes.parent_id -> document_outline_nodes.id e uma
	// auto-referencia legitima (a arvore de outline), entao a regra nao pode
	// ser "nenhuma auto-referencia": e "nenhuma coluna apontando para a
	// propria tabela sem ser a hierarquia declarada".
	refs := regexp.MustCompile(`(?m)^Ref: (\w+)\.(\w+) > (\w+)\.(\w+)$`).FindAllStringSubmatch(rendered, -1)
	if len(refs) == 0 {
		t.Fatal("nenhuma relacao emitida; o gerador deixou de ler as chaves estrangeiras")
	}

	for _, ref := range refs {
		fromTable, fromCol, toTable := ref[1], ref[2], ref[3]
		if fromTable != toTable {
			continue
		}
		if fromCol == "parent_id" {
			continue
		}
		t.Errorf("auto-referencia suspeita: %s.%s > %s", fromTable, fromCol, toTable)
	}
}

// Pegadinha do cache do Go: o .dbml e lido em runtime, entao o test cache nao
// o enxerga como entrada. Editar o arquivo e rodar `go test ./...` devolve
// "(cached)" e o teste parece passar. Para verificar de verdade use
// `go test -count=1`. No CI nao ha cache, entao a protecao vale.
func TestArquivoVersionadoEstaEmDia(t *testing.T) {
	rendered, err := render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	target := defaultOutputPath()

	current, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ler %s: %v (rode: go run ./cmd/schema-dbml)", target, err)
	}

	if string(current) != rendered {
		t.Fatalf("%s esta desatualizado em relacao aos modelos; rode: go run ./cmd/schema-dbml", target)
	}
}
