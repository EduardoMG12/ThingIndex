package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Provedores ecoam a credencial no erro (OpenAI devolve parte da chave, AWS o
// access key id, Ollama a URL com usuario e senha). O erro cru vai para o log;
// providerConnectionFailureResponse tem de responder com mensagem fixa.

func lerFonteDoHandler(t *testing.T, arquivo string) string {
	t.Helper()

	conteudo, err := os.ReadFile(arquivo)
	if err != nil {
		t.Fatalf("nao foi possivel ler %s: %v", arquivo, err)
	}
	return string(conteudo)
}

func TestRespostaDeFalhaDeProviderNaoUsaAMensagemCrua(t *testing.T) {
	fonte := lerFonteDoHandler(t, "provider.go")

	inicio := strings.Index(fonte, "func providerConnectionFailureResponse(")
	if inicio < 0 {
		t.Fatal("providerConnectionFailureResponse nao existe mais: se foi renomeada, " +
			"atualize este guard em vez de remove-lo")
	}

	fim := strings.Index(fonte[inicio:], "\nfunc ")
	if fim < 0 {
		fim = len(fonte) - inicio
	}
	corpo := fonte[inicio : inicio+fim]

	// err.Error() aqui vazaria a credencial na resposta.
	perigosos := []struct {
		padrao *regexp.Regexp
		porque string
	}{
		{
			regexp.MustCompile(`err\.Error\(\)`),
			"err.Error() carrega a mensagem crua do provedor, que ecoa a credencial",
		},
		{
			regexp.MustCompile(`fmt\.Sprintf\([^)]*%v[^)]*,\s*err\b`),
			"formatar o erro com %v embute a credencial na resposta",
		},
	}

	for _, caso := range perigosos {
		if caso.padrao.MatchString(corpo) {
			t.Errorf("providerConnectionFailureResponse expoe o erro cru ao cliente.\n\n"+
				"Motivo: %s.\n\n"+
				"Exemplo real: a OpenAI responde \"Incorrect API key provided: sk-proj-ABC123...\" "+
				"e a AWS inclui o AWSAccessKeyId na falha de assinatura.\n\n"+
				"O detalhe pertence ao log do servidor; a resposta HTTP deve usar i18n.T.",
				caso.porque)
		}
	}
}

func TestTodoRamoDeFalhaDeProviderUsaMensagemTraduzida(t *testing.T) {
	fonte := lerFonteDoHandler(t, "provider.go")

	inicio := strings.Index(fonte, "func providerConnectionFailureResponse(")
	if inicio < 0 {
		t.Skip("funcao renomeada; o guard acima ja acusa")
	}
	fim := strings.Index(fonte[inicio:], "\nfunc ")
	corpo := fonte[inicio : inicio+fim]

	// Todo return deve vir de i18n.T ou de um helper, nunca de string literal.
	retornos := regexp.MustCompile(`return\s+http\.Status\w+,\s+([^\n]+)`).FindAllStringSubmatch(corpo, -1)
	if len(retornos) == 0 {
		t.Fatal("nenhum retorno encontrado: o guard perdeu a capacidade de verificar")
	}

	for _, retorno := range retornos {
		expressao := strings.TrimSpace(retorno[1])
		usaTraducao := strings.Contains(expressao, "i18n.T(") ||
			strings.Contains(expressao, "Message(lang")
		if !usaTraducao {
			t.Errorf("retorno sem mensagem traduzida: %q\n\n"+
				"Toda resposta de falha de provider precisa vir de i18n.T: mensagem "+
				"escrita a mao escapa da traducao e pode carregar detalhe do erro.",
				expressao)
		}
	}
}
