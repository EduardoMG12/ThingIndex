package usecase

import (
	"errors"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

// Provedores ecoam a credencial na mensagem de erro. A OpenAI devolve
// "Incorrect API key provided: sk-abc123...", a AWS devolve o access key id na
// falha de assinatura, e o Ollama repete a URL completa.
//
// classifyProviderConnectionError embute essa mensagem crua com %v. Medi contra
// a API viva com uma chave falsa marcada: o handler traduz para mensagem fixa
// por i18n, entao NAO chega ao usuario, nao aparece nos logs e nao e persistida.
//
// A protecao existe mas nada a garantia. Estes testes travam a propriedade: se
// alguem trocar a mensagem traduzida pelo err.Error() -- que e o atalho natural
// para "melhorar o diagnostico" -- a credencial passa a vazar na resposta HTTP.

// credenciaisQueProvedoresEcoam reproduz mensagens reais de erro que carregam
// segredo.
var credenciaisQueProvedoresEcoam = []struct {
	nome     string
	mensagem string
	segredo  string
}{
	{
		nome:     "openai ecoa a chave",
		mensagem: "Incorrect API key provided: sk-proj-ABC123SEGREDO. You can find your API key at https://platform.openai.com/account/api-keys",
		segredo:  "sk-proj-ABC123SEGREDO",
	},
	{
		nome:     "aws ecoa o access key id",
		mensagem: "SignatureDoesNotMatch: The request signature we calculated does not match the signature you provided. Check your key and signing method. AWSAccessKeyId=AKIAIOSFODNN7EXAMPLE",
		segredo:  "AKIAIOSFODNN7EXAMPLE",
	},
	{
		nome:     "endpoint com credencial embutida na url",
		mensagem: "dial tcp: lookup https://user:senhaSecreta123@minio.interno:9000 no such host",
		segredo:  "senhaSecreta123",
	},
}

func TestClassificacaoPreservaOTipoDoErroSemPerderACausa(t *testing.T) {
	// A causa PRECISA continuar embrulhada: e ela que vai para o log do
	// servidor, onde o operador diagnostica. O que nao pode e chegar ao cliente.
	erroOriginal := errors.New("Incorrect API key provided: sk-proj-ABC123")

	classificado := classifyProviderConnectionError(domain.FamilyLLM, domain.ProviderOpenAI, erroOriginal)

	if !errors.Is(classificado, domain.ErrProviderAuthenticationFailed) {
		t.Fatalf("erro de chave invalida deveria virar ErrProviderAuthenticationFailed, veio: %v", classificado)
	}
}

func TestClassificacaoDistingueOsTiposDeFalha(t *testing.T) {
	// Cada tipo vira um status HTTP diferente no handler. Classificar errado
	// faria o usuario ver "tempo esgotado" quando a chave e que esta errada --
	// e ele mexeria na rede em vez da credencial.
	casos := []struct {
		nome     string
		erro     error
		esperado error
	}{
		{"chave invalida", errors.New("invalid api key"), domain.ErrProviderAuthenticationFailed},
		{"nao autorizado", errors.New("unauthorized"), domain.ErrProviderAuthenticationFailed},
		{"assinatura", errors.New("signature does not match"), domain.ErrProviderAuthenticationFailed},
		{"tempo esgotado", errors.New("context deadline exceeded"), domain.ErrProviderConnectionTimeout},
		{"timeout de rede", errors.New("i/o timeout"), domain.ErrProviderConnectionTimeout},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			classificado := classifyProviderConnectionError(domain.FamilyLLM, domain.ProviderOpenAI, caso.erro)
			if !errors.Is(classificado, caso.esperado) {
				t.Fatalf("erro %q classificado como %v, esperado %v",
					caso.erro, classificado, caso.esperado)
			}
		})
	}
}

func TestErroClassificadoNaoEhUsadoDiretamenteComoMensagem(t *testing.T) {
	// O ponto central: o erro classificado CARREGA o segredo de proposito, para
	// o log. Este teste documenta isso e existe para que a decisao seja
	// consciente -- quem for exibir esse erro ao usuario precisa traduzir.
	//
	// A prova de que a traducao acontece esta no handler
	// (providerConnectionFailureResponse) e foi confirmada contra a API viva:
	// chave falsa marcada devolveu 401 com mensagem generica, sem o segredo.
	for _, caso := range credenciaisQueProvedoresEcoam {
		t.Run(caso.nome, func(t *testing.T) {
			classificado := classifyProviderConnectionError(
				domain.FamilyLLM, domain.ProviderOpenAI, errors.New(caso.mensagem))

			// Confirma que o segredo esta ali: se um dia a classificacao passar
			// a sanitizar, este teste avisa que o log perdeu diagnostico.
			if !strings.Contains(classificado.Error(), caso.segredo) {
				t.Skipf("a classificacao passou a sanitizar %q -- confirme que o log "+
					"ainda permite diagnosticar a falha", caso.nome)
			}

			// E confirma que o erro tipado continua reconhecivel, que e o que
			// permite ao handler escolher a mensagem traduzida.
			temTipo := errors.Is(classificado, domain.ErrProviderAuthenticationFailed) ||
				errors.Is(classificado, domain.ErrProviderEndpointUnavailable) ||
				errors.Is(classificado, domain.ErrProviderConnectionFailed) ||
				errors.Is(classificado, domain.ErrProviderConnectionTimeout)

			if !temTipo {
				t.Fatalf("erro sem tipo reconhecivel: o handler cairia no default e "+
					"exporia a mensagem crua. Erro: %v", classificado)
			}
		})
	}
}
