package evalcorpus

import (
	"regexp"
	"strings"
)

// numeroDecimal casa um numero com parte fracionaria escrito com ponto, fora
// de comandos LaTeX. Os limites de palavra impedem mexer em versoes
// ("0.5.0") so quando o numero tem exatamente um ponto entre digitos.
var numeroDecimal = regexp.MustCompile(`(^|[^0-9.])(-?[0-9]+)\.([0-9]+)($|[^0-9.])`)

// VirgulaDecimal troca o separador decimal de ponto para virgula, como pede a
// norma do portugues, nas tabelas que os avaliadores geram para o artigo. A
// virgula vai entre chaves ({,}) porque, em modo matematico, o LaTeX poe
// espaco depois da virgula e "0,5" viraria "0, 5".
//
// Um unico ponto de conversao, aplicado na escrita do .tex, evita espalhar a
// formatacao por cada Fprintf e mantem o calculo e a saida de terminal com o
// ponto de sempre.
func VirgulaDecimal(tex string) string {
	// Duas passadas: numeros adjacentes ("0.1 0.2") dividem o separador, e a
	// primeira passada consome o caractere entre eles.
	for range 2 {
		tex = numeroDecimal.ReplaceAllString(tex, "${1}${2}{,}${3}${4}")
	}
	return tex
}

// RegrasBooktabs troca os \hline de cada tabular pelas regras do pacote
// booktabs (\toprule, \midrule, \bottomrule), que as tabelas escritas a
// mao no artigo ja usam: a primeira regra de cada tabular vira \toprule, a
// ultima \bottomrule e as do meio \midrule.
func RegrasBooktabs(tex string) string {
	linhas := strings.Split(tex, "\n")
	var abertas []int
	for i, linha := range linhas {
		t := strings.TrimSpace(linha)
		switch {
		case t == `\hline`:
			abertas = append(abertas, i)
		case strings.HasPrefix(t, `\end{tabular}`), strings.HasPrefix(t, `\end{longtable}`):
			for j, idx := range abertas {
				regra := `\midrule`
				if j == 0 {
					regra = `\toprule`
				} else if j == len(abertas)-1 {
					regra = `\bottomrule`
				}
				linhas[idx] = strings.Replace(linhas[idx], `\hline`, regra, 1)
			}
			abertas = nil
		}
	}
	return strings.Join(linhas, "\n")
}

// sinalNumero casa um hifen usado como sinal de numero em celula de tabela
// (precedido de espaco ou '&', seguido de digito), fora de modo matematico.
var sinalNumero = regexp.MustCompile(`(^|[ &])-([0-9])`)

// SinalMenos troca o hifen usado como sinal negativo nas celulas pelo sinal de
// menos tipografico ($-$). Numero ja em modo matematico ($-0{,}1$) e palavras
// com hifen (top-10) nao casam.
func SinalMenos(tex string) string {
	return sinalNumero.ReplaceAllString(tex, "${1}$$-$$${2}")
}

// Tabela aplica as convencoes do artigo a um .tex gerado: virgula decimal,
// regras do booktabs e sinal de menos.
func Tabela(tex string) string {
	return SinalMenos(RegrasBooktabs(VirgulaDecimal(tex)))
}
