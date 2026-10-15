package usecase

import (
	"testing"

	"thing-index/internal/domain"
)

// POST /user/providers/models aceita credentials do corpo e a descoberta do
// Ollama dispensa api_key, entao qualquer usuario aponta base_url para um
// servidor proprio. Essa gravacao nao pode cair no escopo GLOBAL.
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

		// Provider com credencial propria: catalogo por usuario mesmo no caminho
		// confiavel.
		scope, scopedUserID := modelCatalogScope(atacante, domain.ProviderOpenAI, true)

		if scope != "USER:42" {
			t.Fatalf("provider privado precisa ficar no escopo do usuario, veio %q", scope)
		}
		if scopedUserID == nil || *scopedUserID != atacante {
			t.Fatal("provider privado precisa gravar o user_id na linha")
		}
	})
}
