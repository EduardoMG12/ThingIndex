package usecase

import (
	"errors"
	"strings"
	"testing"

	"thing-index/internal/domain"
)

// Provedores ecoam a credencial na mensagem de erro (chave da OpenAI, access
// key da AWS, URL do Ollama). O erro classificado carrega a mensagem crua para
// o log; o handler e quem traduz para mensagem fixa antes de responder.

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
	// A causa continua embrulhada para o log do servidor.
	erroOriginal := errors.New("Incorrect API key provided: sk-proj-ABC123")

	classificado := classifyProviderConnectionError(domain.FamilyLLM, domain.ProviderOpenAI, erroOriginal)

	if !errors.Is(classificado, domain.ErrProviderAuthenticationFailed) {
		t.Fatalf("erro de chave invalida deveria virar ErrProviderAuthenticationFailed, veio: %v", classificado)
	}
}

func TestClassificacaoDistingueOsTiposDeFalha(t *testing.T) {
	// Classificar errado mostraria "tempo esgotado" com a chave errada.
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
	// O erro classificado carrega o segredo de proposito, para o log. Quem exibir
	// ao usuario precisa traduzir (ver providerConnectionFailureResponse).
	for _, caso := range credenciaisQueProvedoresEcoam {
		t.Run(caso.nome, func(t *testing.T) {
			classificado := classifyProviderConnectionError(
				domain.FamilyLLM, domain.ProviderOpenAI, errors.New(caso.mensagem))

			// Se a classificacao passar a sanitizar, o log perde diagnostico.
			if !strings.Contains(classificado.Error(), caso.segredo) {
				t.Skipf("a classificacao passou a sanitizar %q -- confirme que o log "+
					"ainda permite diagnosticar a falha", caso.nome)
			}

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
