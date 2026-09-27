package evalcorpus

import "testing"

func TestVirgulaDecimal(t *testing.T) {
	casos := []struct{ entrada, esperado string }{
		{"0.750", "0{,}750"},
		{`\shortstack{0.750 \\ \scriptsize[0.500, 0.938]}`, `\shortstack{0{,}750 \\ \scriptsize[0{,}500, 0{,}938]}`},
		{"-0.429 & 1.000", "-0{,}429 & 1{,}000"},
		{"48.4\\%", "48{,}4\\%"},
		{"pgvector 0.5.0", "pgvector 0.5.0"},
		{"Tabela 3 e $k_1$", "Tabela 3 e $k_1$"},
		{"1.4e-16", "1{,}4e-16"},
	}
	for _, c := range casos {
		if got := VirgulaDecimal(c.entrada); got != c.esperado {
			t.Errorf("VirgulaDecimal(%q) = %q, quero %q", c.entrada, got, c.esperado)
		}
	}
}
