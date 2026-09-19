package indexing

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// O prompt gerador de perguntas esta congelado pelo pre-registro em
// docs/prds/pre-registro-perguntas-geradas.md.
//
// Este teste existe para tornar a alteracao VISIVEL, nao para proibi-la. O
// compromisso registrado e rodar uma versao so: rodar varias e reportar a
// melhor produz um numero favoravel e sem valor, e sem um hash ninguem
// consegue detectar isso depois, nem o proprio autor daqui a um mes.
//
// Se o prompt precisar mudar, mude tambem este hash NO MESMO commit, e escreva
// um pre-registro novo. A medicao anterior passa a nao valer.
const hashDoPromptDePerguntas = "12c07448ee11a62ea87181cf708f13360139e4418d0b7e3952a1a4977f50be21"

func TestPromptDePerguntasEstaCongelado(t *testing.T) {
	t.Parallel()

	sistema, usuario := buildPageQuestionsPrompt("")
	soma := sha256.Sum256([]byte(sistema + "\n---\n" + usuario))
	atual := hex.EncodeToString(soma[:])

	if atual != hashDoPromptDePerguntas {
		t.Fatalf(`o prompt gerador de perguntas mudou.

Se a mudanca for intencional: atualize hashDoPromptDePerguntas para %q NESTE
commit e escreva um pre-registro novo em docs/prds/. A medicao registrada em
docs/prds/pre-registro-perguntas-geradas.md passa a nao valer, porque ela
supoe este prompt exato.`, atual)
	}
}

// O prompt de perguntas precisa pedir o OPOSTO do prompt de resumo. Se um dia
// alguem "melhorar" o prompt de perguntas pedindo os numeros e termos da
// pagina, ele vira o resumo de novo e o experimento perde o sentido sem que
// nenhum teste reclame.
func TestPromptDePerguntasPedeLinguagemDeLeigo(t *testing.T) {
	t.Parallel()

	sistema, usuario := buildPageQuestionsPrompt("")
	juntos := strings.ToLower(sistema + " " + usuario)

	for _, exigido := range []string{"everyday", "non-specialist", "do not reuse the technical vocabulary"} {
		if !strings.Contains(juntos, exigido) {
			t.Errorf("o prompt perdeu a instrucao %q, que e o que o distingue do resumo", exigido)
		}
	}

	resumoSistema, resumoUsuario := buildPageSummaryPrompt("")
	if sistema+usuario == resumoSistema+resumoUsuario {
		t.Fatal("o prompt de perguntas ficou identico ao de resumo")
	}
}
