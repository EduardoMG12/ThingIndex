# Corpus de licitações: a lei revogada e a que a revogou

Dois documentos escolhidos porque respondem **diferente** para a mesma pergunta,
e porque a divergência tem data: o Art. 193 da Lei 14.133/2021 revoga a Lei
8.666/1993, e o Art. 190 manda que contrato assinado antes continue regido pela
lei revogada. É o caso em que "qual regra vale" depende de quando.

| Arquivo | Origem | Páginas |
|---|---|---|
| `lei-8666-1993-licitacoes.pdf` | Planalto, texto compilado: `https://www.planalto.gov.br/ccivil_03/leis/l8666cons.htm` | 52 |
| `lei-14133-2021-licitacoes.pdf` | Planalto, texto compilado: `https://www.planalto.gov.br/ccivil_03/_ato2019-2022/2021/lei/l14133.htm` | 77 |

## Como os PDFs foram gerados

O Planalto publica o texto compilado em HTML, não em PDF, e o Senado bloqueia
download automático. Os arquivos aqui são conversão fiel do HTML oficial:

```sh
curl -sL -A "Mozilla/5.0" -o l14133.htm \
  "https://www.planalto.gov.br/ccivil_03/_ato2019-2022/2021/lei/l14133.htm"
# o HTML do Planalto vem em latin-1 e sem declaração de charset
python3 -c "raw=open('l14133.htm','rb').read(); t=raw.decode('latin-1'); \
  t=t.replace('<head>','<head><meta charset=\"utf-8\">',1); \
  open('l14133.utf8.html','w',encoding='utf-8').write(t)"
chromium --headless --disable-gpu --print-to-pdf=l14133.pdf file://$PWD/l14133.utf8.html
```

O texto compilado é o que vale para este corpus: ele traz as redações sucessivas
de cada artigo, com as marcações de vigência. A página 75 da Lei 14.133 mostra o
Art. 191 três vezes (redação original, redação da Medida Provisória 1.167/2023
com "vigência encerrada", e redação da Lei Complementar 198/2023), que é
exatamente o tipo de armadilha que o corpus existe para exercitar.

## Âncoras verificadas com pdftotext

| Fato | Onde |
|---|---|
| Modalidades de licitação (concorrência, tomada de preços, convite, concurso, leilão) | 8.666, Art. 22, p. 16 |
| Modalidades de licitação (pregão, concorrência, concurso, leilão, diálogo competitivo) | 14.133, Art. 28, p. 18 |
| Dispensa até R$ 100.000 em obras e serviços de engenharia | 14.133, Art. 75, I, p. 36 |
| Dispensa até R$ 50.000 em outros serviços e compras | 14.133, Art. 75, II, p. 36 |
| Revogação da Lei 8.666 em 30/12/2023 | 14.133, Art. 193, II, p. 75 |
| Contrato assinado antes segue a lei revogada | 14.133, Art. 190, p. 75 |
| Marcação "Revogado pela Lei nº 14.133, de 2021" no cabeçalho | 8.666, p. 1 |

## Ampliação de 2026-09-19

Acrescentada a `lei-10520-pregao.pdf` (Lei do Pregão, 4 p.), que integra a mesma
cadeia de revogação: o art. 193, II, da Lei 14.133 revoga a Lei 8.666 **e** a Lei
10.520 na mesma data. O acervo passa a ter duas normas revogadas e uma vigente,
o que torna a pergunta "qual regra vale" menos trivial que com dois documentos.
