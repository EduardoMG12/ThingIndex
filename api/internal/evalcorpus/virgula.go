package evalcorpus

import "regexp"

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
