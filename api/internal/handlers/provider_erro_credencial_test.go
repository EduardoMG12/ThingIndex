package handlers

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Provedores ecoam a credencial na mensagem de erro: a OpenAI devolve
// "Incorrect API key provided: sk-abc123...", a AWS devolve o access key id, e
// o Ollama repete a URL completa -- que pode conter usuario e senha.
//
// classifyProviderConnectionError embute essa mensagem crua no erro DE
// PROPOSITO, porque e ela que vai para o log e permite ao operador
// diagnosticar. A protecao esta em providerConnectionFailureResponse, que
// traduz para mensagem fixa antes de responder ao cliente.
//
// Confirmado contra a API viva: chave falsa marcada devolveu 401 com mensagem
// generica, sem o segredo na resposta, nos logs ou no banco.
//
// O risco e a "melhoria" natural: alguem achar a mensagem generica pouco util e
// trocar por err.Error() para ajudar o usuario a diagnosticar. A credencial
// passaria a vazar na resposta HTTP sem que nenhum teste existente acusasse.

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

	// err.Error() ou %v aplicado ao erro dentro desta funcao significaria expor
	// a mensagem do provedor -- com a credencial dentro -- na resposta HTTP.
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

	// Cada `return` da funcao precisa devolver uma mensagem vinda de i18n.T ou
	// de um helper dedicado. Retorno com string literal indicaria mensagem
	// escrita a mao, que nao passa por traducao e pode ter sido copiada do erro.
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
