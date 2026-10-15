package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Falha se um filtro user_id sumir de uma query de leitura. Verificacao
// estrutural, lendo o fonte, para rodar sem Postgres.

// Recursos acessiveis por id vindo da URL: sem o filtro, trocar o id expoe outra conta.
var metodosQuePrecisamDeDono = []struct {
	arquivo string
	metodo  string
	porque  string
}{
	{"gorm_catalog_document_repository.go", "GetDocumentByID", "documento acessado por id na URL"},
	{"gorm_collection_repository.go", "GetCollectionByID", "colecao acessada por id na URL"},
	{"gorm_collection_repository.go", "GetCollectionByNormalizedName", "colecao resolvida por nome no upload"},
	{"gorm_collection_repository.go", "ListCollectionsByUserID", "listagem do acervo"},
	{"gorm_category_repository.go", "GetCategoryByID", "categoria acessada por id"},
}

func lerFonteDoRepositorio(t *testing.T, arquivo string) string {
	t.Helper()

	conteudo, err := os.ReadFile(filepath.Join(".", arquivo))
	if err != nil {
		t.Fatalf("nao foi possivel ler %s: %v", arquivo, err)
	}
	return string(conteudo)
}

// corpoDoMetodo devolve o texto entre a assinatura e o proximo `\nfunc `.
func corpoDoMetodo(fonte, metodo string) (string, bool) {
	marcador := ") " + metodo + "("
	inicio := strings.Index(fonte, marcador)
	if inicio < 0 {
		return "", false
	}

	fim := strings.Index(fonte[inicio:], "\nfunc ")
	if fim < 0 {
		return fonte[inicio:], true
	}
	return fonte[inicio : inicio+fim], true
}

func TestLeiturasPorIDFiltramPeloDono(t *testing.T) {
	for _, caso := range metodosQuePrecisamDeDono {
		t.Run(caso.metodo, func(t *testing.T) {
			fonte := lerFonteDoRepositorio(t, caso.arquivo)

			corpo, encontrado := corpoDoMetodo(fonte, caso.metodo)
			if !encontrado {
				t.Fatalf("metodo %s nao existe mais em %s: se foi renomeado, "+
					"atualize esta lista em vez de remover a verificacao",
					caso.metodo, caso.arquivo)
			}

			if !strings.Contains(corpo, "user_id") {
				t.Fatalf("%s.%s nao filtra por user_id.\n\nMotivo do filtro: %s.\n\n"+
					"Sem ele, trocar o id na URL devolve dado de outra conta.",
					caso.arquivo, caso.metodo, caso.porque)
			}
		})
	}
}

// TestNenhumaQueryUsaFirstSemCondicao protege contra outra forma do mesmo
// defeito: First() sem Where devolve a PRIMEIRA linha da tabela, que quase
// sempre pertence a outro usuario.
func TestNenhumaQueryUsaFirstSemCondicao(t *testing.T) {
	arquivos, err := filepath.Glob("gorm_*.go")
	if err != nil {
		t.Fatalf("glob falhou: %v", err)
	}

	// O perigoso e First logo apos WithContext, sem Where.
	perigoso := regexp.MustCompile(`WithContext\(ctx\)\.\s*First\(`)

	for _, arquivo := range arquivos {
		if strings.HasSuffix(arquivo, "_test.go") {
			continue
		}

		conteudo := lerFonteDoRepositorio(t, arquivo)
		if perigoso.MatchString(conteudo) {
			t.Errorf("%s tem First() sem condicao: devolveria a primeira linha da "+
				"tabela, que provavelmente e de outro usuario", arquivo)
		}
	}
}

func TestDelecaoExigeDono(t *testing.T) {
	arquivos, err := filepath.Glob("gorm_*.go")
	if err != nil {
		t.Fatalf("glob falhou: %v", err)
	}

	for _, arquivo := range arquivos {
		if strings.HasSuffix(arquivo, "_test.go") {
			continue
		}

		fonte := lerFonteDoRepositorio(t, arquivo)

		for _, metodo := range []string{"DeleteDocument", "DeleteCollection", "DeleteCategory"} {
			corpo, encontrado := corpoDoMetodo(fonte, metodo)
			if !encontrado {
				continue
			}
			if !strings.Contains(corpo, "user_id") {
				t.Errorf("%s.%s apaga sem verificar o dono", arquivo, metodo)
			}
		}
	}
}
