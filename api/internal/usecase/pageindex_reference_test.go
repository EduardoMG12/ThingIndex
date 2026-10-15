package usecase

import "testing"

// Travam as propriedades da fixture de referencia do PageIndex usada no
// benchmark de indexacao.

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

	if fixture.SourcePath != "pageindex/results" {
		t.Fatalf("origem da referencia mudou para %q; o artigo afirma pageindex/results",
			fixture.SourcePath)
	}
}

// Um dos artefatos tem intervalo de pagina invertido; a comparacao e barra de
// qualidade, nao gabarito.
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
