// Command schema-dbml gera o esquema relacional do ThingIndex em DBML.
//
// Motivo: o Apendice A do artigo promete "o esquema completo em formato DBML no
// repositorio do projeto", e ate agora nenhum arquivo .dbml existia -- a
// promessa era falsa e verificavel em um `find`. Alem disso o artigo afirmava
// 20 tabelas, quando o AutoMigrate registra 21.
//
// Escrever o .dbml a mao so trocaria um numero desatualizado por outro. O
// gerador le os mesmos modelos que db.ManagedModels() entrega ao AutoMigrate e
// usa o parser de schema do proprio GORM, entao o documento do apendice e
// derivado do esquema real em vez de descrever de memoria o que se acredita que
// ele seja. Se um modelo mudar e o .dbml nao for regerado, o CI acusa
// (`make schema-dbml-check`).
//
// Uso:
//
//	go run ./cmd/schema-dbml            # escreve docs/schema/thingindex.dbml
//	go run ./cmd/schema-dbml -check     # falha se o arquivo estiver desatualizado
//	go run ./cmd/schema-dbml -o -       # emite em stdout
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	appdb "thing-index/db"

	"gorm.io/gorm/schema"
)

// defaultOutputPath resolve o destino a partir da localizacao deste arquivo-fonte,
// nao do diretorio de trabalho. Sem isso o caminho relativo funcionava em
// `go run ./cmd/schema-dbml` (cwd=api/) e quebrava em `go test ./...`, que roda
// com o cwd no diretorio do pacote -- o teste falhava por nao achar o arquivo,
// nao por o esquema estar errado.
func defaultOutputPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "docs/schema/thingindex.dbml"
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(repoRoot, "docs", "schema", "thingindex.dbml")
}

func main() {
	output := flag.String("o", defaultOutputPath(), "caminho do arquivo de saida, ou - para stdout")
	check := flag.Bool("check", false, "nao escreve; sai com codigo 1 se o arquivo estiver desatualizado")
	flag.Parse()

	rendered, err := render()
	if err != nil {
		fmt.Fprintf(os.Stderr, "schema-dbml: %v\n", err)
		os.Exit(1)
	}

	if *output == "-" {
		fmt.Print(rendered)
		return
	}

	if *check {
		current, err := os.ReadFile(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "schema-dbml: %s nao pode ser lido: %v\n", *output, err)
			fmt.Fprintf(os.Stderr, "rode: go run ./cmd/schema-dbml\n")
			os.Exit(1)
		}
		if string(current) != rendered {
			fmt.Fprintf(os.Stderr, "schema-dbml: %s esta desatualizado em relacao aos modelos\n", *output)
			fmt.Fprintf(os.Stderr, "rode: go run ./cmd/schema-dbml\n")
			os.Exit(1)
		}
		fmt.Printf("schema-dbml: %s esta em dia\n", *output)
		return
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "schema-dbml: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, []byte(rendered), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "schema-dbml: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("schema-dbml: %s escrito\n", *output)
}

type relation struct {
	fromTable string
	fromCol   string
	toTable   string
	toCol     string
}

func render() (string, error) {
	var (
		cache   sync.Map
		naming  = schema.NamingStrategy{}
		tables  []*schema.Schema
		byModel = map[string]*schema.Schema{}
	)

	for _, model := range appdb.ManagedModels() {
		parsed, err := schema.Parse(model, &cache, naming)
		if err != nil {
			return "", fmt.Errorf("parse do modelo %T: %w", model, err)
		}
		tables = append(tables, parsed)
		byModel[parsed.Name] = parsed
	}

	var b strings.Builder
	b.WriteString("// Esquema relacional do ThingIndex -- Apendice A do artigo.\n")
	b.WriteString("//\n")
	b.WriteString("// ARQUIVO GERADO. Nao edite a mao.\n")
	b.WriteString("// Fonte: api/db/models.go (a mesma lista que alimenta o AutoMigrate).\n")
	b.WriteString("// Regerar: cd api && go run ./cmd/schema-dbml\n")
	b.WriteString("//\n")
	b.WriteString("// Importavel em https://dbdiagram.io para visualizacao interativa.\n")
	b.WriteString(fmt.Sprintf("//\n// Total: %d tabelas.\n\n", len(tables)))

	var relations []relation

	for _, table := range tables {
		b.WriteString(fmt.Sprintf("Table %s {\n", table.Table))

		for _, field := range table.Fields {
			if field.DBName == "" {
				// Campo apenas de aplicacao (gorm:"-") ou associacao pura:
				// nao vira coluna.
				continue
			}

			var attrs []string
			if field.PrimaryKey {
				attrs = append(attrs, "pk")
				if field.AutoIncrement {
					attrs = append(attrs, "increment")
				}
			}
			if field.NotNull && !field.PrimaryKey {
				attrs = append(attrs, "not null")
			}
			if field.Unique {
				attrs = append(attrs, "unique")
			}
			if field.HasDefaultValue && field.DefaultValue != "" {
				attrs = append(attrs, fmt.Sprintf("default: `%s`", field.DefaultValue))
			}

			line := fmt.Sprintf("  %s %s", field.DBName, dbmlType(field))
			if len(attrs) > 0 {
				line += fmt.Sprintf(" [%s]", strings.Join(attrs, ", "))
			}
			b.WriteString(line + "\n")
		}

		if idx := renderIndexes(table); idx != "" {
			b.WriteString("\n" + idx)
		}

		b.WriteString("}\n\n")

		relations = append(relations, collectRelations(table)...)
	}

	if len(relations) > 0 {
		b.WriteString("// Relacionamentos (chaves estrangeiras declaradas nos modelos GORM)\n\n")
		sort.Slice(relations, func(i, j int) bool {
			if relations[i].fromTable != relations[j].fromTable {
				return relations[i].fromTable < relations[j].fromTable
			}
			return relations[i].fromCol < relations[j].fromCol
		})
		seen := map[string]bool{}
		for _, rel := range relations {
			line := fmt.Sprintf("Ref: %s.%s > %s.%s\n", rel.fromTable, rel.fromCol, rel.toTable, rel.toCol)
			if seen[line] {
				continue
			}
			seen[line] = true
			b.WriteString(line)
		}
	}

	return b.String(), nil
}

func renderIndexes(table *schema.Schema) string {
	indexes := table.ParseIndexes()
	if len(indexes) == 0 {
		return ""
	}

	sorted := make([]*schema.Index, len(indexes))
	copy(sorted, indexes)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var b strings.Builder
	b.WriteString("  indexes {\n")
	for _, idx := range sorted {
		cols := make([]string, 0, len(idx.Fields))
		for _, f := range idx.Fields {
			cols = append(cols, f.DBName)
		}
		if len(cols) == 0 {
			continue
		}

		attrs := []string{fmt.Sprintf("name: %q", idx.Name)}
		if strings.EqualFold(idx.Class, "UNIQUE") {
			attrs = append(attrs, "unique")
		}

		target := cols[0]
		if len(cols) > 1 {
			target = "(" + strings.Join(cols, ", ") + ")"
		}
		b.WriteString(fmt.Sprintf("    %s [%s]\n", target, strings.Join(attrs, ", ")))
	}
	b.WriteString("  }\n")
	return b.String()
}

// collectRelations extrai as chaves estrangeiras de um schema.
//
// Cuidado que custou um bug: o GORM registra a relacao nos DOIS schemas. Ao
// parsear ChatConversation (HasMany Messages), a mesma relacao aparece tambem
// em ChatConversationEntry. Deduzir a tabela a partir de `table` ou de
// `rel.FieldSchema` produzia auto-referencias falsas do tipo
// `chat_conversation_entries.conversation_id > chat_conversation_entries.id`.
//
// A leitura confiavel e ignorar quem declarou a relacao e perguntar aos
// proprios campos a que schema pertencem: a FK sempre mora na tabela "muitos" e
// a PK na tabela "um", independente do lado em que a associacao foi escrita.
func collectRelations(table *schema.Schema) []relation {
	var out []relation

	for _, rel := range table.Relationships.Relations {
		if rel.FieldSchema == nil || rel.Type == schema.Many2Many {
			continue
		}

		for _, ref := range rel.References {
			if ref.PrimaryKey == nil || ref.ForeignKey == nil {
				continue
			}
			if ref.PrimaryKey.Schema == nil || ref.ForeignKey.Schema == nil {
				continue
			}

			out = append(out, relation{
				fromTable: ref.ForeignKey.Schema.Table,
				fromCol:   ref.ForeignKey.DBName,
				toTable:   ref.PrimaryKey.Schema.Table,
				toCol:     ref.PrimaryKey.DBName,
			})
		}
	}

	return out
}

// dbmlType traduz o tipo do GORM para um nome reconhecido pelo dbdiagram.io.
// Prioriza a anotacao explicita `gorm:"type:..."` quando ela existe, porque e
// ela que descreve o tipo real criado no Postgres (jsonb, text, etc.).
func dbmlType(field *schema.Field) string {
	if raw := strings.TrimSpace(strings.ToLower(string(field.DataType))); raw != "" {
		switch raw {
		case "string":
			if field.Size > 0 {
				return fmt.Sprintf("varchar(%d)", field.Size)
			}
			return "text"
		case "int", "uint":
			if field.AutoIncrement {
				return "serial"
			}
			return "integer"
		case "bool":
			return "boolean"
		case "time":
			return "timestamptz"
		case "float":
			return "numeric"
		case "bytes":
			return "bytea"
		default:
			// Tipos vindos de `type:` no tag (jsonb, text, numeric(...)) ja
			// sao validos em Postgres e no dbdiagram.
			return raw
		}
	}
	return "text"
}
