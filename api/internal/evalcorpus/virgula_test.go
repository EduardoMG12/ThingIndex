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

func TestRegrasBooktabs(t *testing.T) {
	entrada := "\\begin{tabular}{lr}\n\\hline\nA & B \\\\\n\\hline\nx & 1 \\\\\n\\hline\n\\end{tabular}"
	esperado := "\\begin{tabular}{lr}\n\\toprule\nA & B \\\\\n\\midrule\nx & 1 \\\\\n\\bottomrule\n\\end{tabular}"
	if got := RegrasBooktabs(entrada); got != esperado {
		t.Fatalf("RegrasBooktabs:\n%s\nesperado:\n%s", got, esperado)
	}
	// Duas tabelas no mesmo texto sao tratadas uma a uma.
	dupla := entrada + "\n" + entrada
	if got := RegrasBooktabs(dupla); got != esperado+"\n"+esperado {
		t.Fatalf("duas tabelas: %s", got)
	}
}

func TestSinalMenos(t *testing.T) {
	casos := map[string]string{
		"& -0{,}114 \\\\": "& $-$0{,}114 \\\\",
		"& +0{,}429 \\\\": "& +0{,}429 \\\\",
		"$-0{,}188$":      "$-0{,}188$",
		"top-10":          "top-10",
	}
	for entrada, esperado := range casos {
		if got := SinalMenos(entrada); got != esperado {
			t.Errorf("SinalMenos(%q) = %q, esperado %q", entrada, got, esperado)
		}
	}
}
