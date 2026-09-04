package usecase

import (
	"testing"

	"thing-index/internal/domain"
)

// A rota autenticada POST /user/providers/models aceita `credentials` do corpo
// da requisicao, e a descoberta de modelos do Ollama e explicitamente isenta de
// api_key (o /api/tags e publico). Isso significa que qualquer usuario logado
// consegue apontar `base_url` para um servidor sob seu controle e fazer o
// backend persistir a lista devolvida por ele.
//
// O defeito estava no destino dessa gravacao: modelCatalogScope descartava o
// userID para providers "publicos" e escrevia no escopo GLOBAL, que e cache
// compartilhado por todos os usuarios da instalacao. Como ReplaceModelCatalog
// desativa todo o escopo antes de inserir, dava para envenenar o seletor de
// modelos de terceiros ou -- devolvendo lista vazia -- zerar o catalogo global
// de todo mundo.
//
// Verificado RED -> GREEN: removendo o parametro `trusted` da condicao (ou seja,
// voltando ao `if isPublicModelCatalogProvider(providerType)`), o primeiro
// subteste falha com escopo "GLOBAL" e userID nil.
func TestModelCatalogScopeNaoDeixaRequisicaoDoClienteEscreverNoEscopoGlobal(t *testing.T) {
	t.Parallel()

	const atacante uint = 42

	t.Run("descoberta vinda de requisicao HTTP fica no escopo do proprio usuario", func(t *testing.T) {
		t.Parallel()

		scope, scopedUserID := modelCatalogScope(atacante, domain.ProviderOllama, false)

		if scope == globalModelCatalogScope {
			t.Fatalf("requisicao de cliente nao pode escrever no escopo compartilhado %q", globalModelCatalogScope)
		}
		if scope != "USER:42" {
			t.Fatalf("esperava escopo do proprio usuario, veio %q", scope)
		}
		if scopedUserID == nil {
			t.Fatal("o user_id precisa ser gravado na linha para o catalogo ficar isolado")
		}
		if *scopedUserID != atacante {
			t.Fatalf("esperava user_id %d, veio %d", atacante, *scopedUserID)
		}
	})

	t.Run("worker de sync continua dono do escopo global", func(t *testing.T) {
		t.Parallel()

		// SyncModelCatalogs chama com userID 0 e o endpoint publico padrao.
		scope, scopedUserID := modelCatalogScope(0, domain.ProviderOllama, true)

		if scope != globalModelCatalogScope {
			t.Fatalf("o sync publico precisa continuar alimentando o cache compartilhado, veio %q", scope)
		}
		if scopedUserID != nil {
			t.Fatalf("a linha global nao pertence a usuario nenhum, veio user_id %d", *scopedUserID)
		}
	})

	t.Run("provider privado nunca cai no escopo global, nem vindo do worker", func(t *testing.T) {
		t.Parallel()

		// Provider com credencial propria: mesmo no caminho confiavel, o
		// catalogo e por usuario -- cada um ve os modelos da sua conta.
		scope, scopedUserID := modelCatalogScope(atacante, domain.ProviderOpenAI, true)

		if scope != "USER:42" {
			t.Fatalf("provider privado precisa ficar no escopo do usuario, veio %q", scope)
		}
		if scopedUserID == nil || *scopedUserID != atacante {
			t.Fatal("provider privado precisa gravar o user_id na linha")
		}
	})
}
