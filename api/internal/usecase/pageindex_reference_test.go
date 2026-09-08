package usecase

import "testing"

// O artigo afirma, em Trabalhos Relacionados, que o benchmark de indexacao
// compara a estrutura produzida pelo indexador contra artefatos do PageIndex --
// citando TRES documentos e a ressalva do intervalo de pagina invertido.
//
// Estes testes travam essas afirmacoes na fixture. Sem eles, alguem que edite o
// artefato de referencia deixa o texto do artigo citando um numero que nao vale
// mais, e nada acusa.

func TestReferenciaDoPageIndexTemOsTresArtefatosCitadosNoArtigo(t *testing.T) {
	fixture, err := loadIndexingBenchmarkReferenceFixture()
	if err != nil {
		t.Fatalf("nao foi possivel carregar a referencia: %v", err)
	}

	const artefatosCitadosNoArtigo = 3
	if len(fixture.Artifacts) != artefatosCitadosNoArtigo {
		t.Fatalf("o artigo cita %d artefatos de referencia, a fixture tem %d",
			artefatosCitadosNoArtigo, len(fixture.Artifacts))
	}

	// A origem tem que continuar sendo o PageIndex: se alguem trocar a fonte, a
	// secao de Trabalhos Relacionados passa a descrever outra ferramenta.
	if fixture.SourcePath != "pageindex/results" {
		t.Fatalf("origem da referencia mudou para %q; o artigo afirma pageindex/results",
			fixture.SourcePath)
	}
}

// TestReferenciaMantemODefeitoQueOArtigoDeclara: o texto afirma que um dos
// artefatos tem intervalo de pagina invertido, e usa isso para justificar por
// que a comparacao e "barra de qualidade" e nao gabarito.
//
// Se o artefato for corrigido, a ressalva do artigo vira afirmacao falsa sobre
// o proprio material -- e o avaliador que abrir o JSON encontra a divergencia.
func TestReferenciaMantemODefeitoQueOArtigoDeclara(t *testing.T) {
	fixture, err := loadIndexingBenchmarkReferenceFixture()
	if err != nil {
		t.Fatalf("nao foi possivel carregar a referencia: %v", err)
	}

	comIntervaloInvalido := 0
	for _, artefato := range fixture.Artifacts {
		if artefato.InvalidRangeCount > 0 {
			comIntervaloInvalido++
		}
	}

	if comIntervaloInvalido == 0 {
		t.Fatal("nenhum artefato tem intervalo invalido, mas o artigo afirma que um deles tem: " +
			"a ressalva de 'barra de qualidade, nao gabarito' perdeu a evidencia")
	}
}

// TestArtefatosDeReferenciaTemPerfilEstrutural garante que a comparacao mede
// algo: artefato sem nos nao serve de referencia para profundidade nem para
// cobertura de resumo.
func TestArtefatosDeReferenciaTemPerfilEstrutural(t *testing.T) {
	fixture, err := loadIndexingBenchmarkReferenceFixture()
	if err != nil {
		t.Fatalf("nao foi possivel carregar a referencia: %v", err)
	}

	for _, artefato := range fixture.Artifacts {
		if artefato.TotalNodes <= 0 {
			t.Fatalf("artefato %q sem nos: nao serve como referencia estrutural", artefato.ID)
		}
		if artefato.MaxDepth <= 0 {
			t.Fatalf("artefato %q sem profundidade declarada", artefato.ID)
		}
		if artefato.SourcePath == "" {
			t.Fatalf("artefato %q sem caminho de origem: a proveniencia da referencia se perde", artefato.ID)
		}
	}
}
