package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// O .dbml nao pode perder tabela, inventar relacao nem ficar defasado.

func TestRenderCobreTodosOsModelosGerenciados(t *testing.T) {
	rendered, err := render()
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	tables := regexp.MustCompile(`(?m)^Table (\w+) \{`).FindAllStringSubmatch(rendered, -1)

	// Conferido contra um Postgres 16 limpo via information_schema.tables.
	const esperado = 25
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

	// O GORM registra HasMany nos dois schemas e a primeira versao gerava refs
	// falsas como chat_conversation_entries.conversation_id >
	// chat_conversation_entries.id. document_outline_nodes.parent_id e
	// auto-referencia legitima (arvore de outline).
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

// O .dbml e lido em runtime, entao o test cache nao o ve como entrada: use
// `go test -count=1` para conferir localmente.
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
