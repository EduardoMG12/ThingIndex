# Artigo do TCC — ThingIndex

Autor: Charles Eduardo Mello Guimarães — IFPR Campus Palmas, Sistemas de Informação.

Título: *ThingIndex: Inteligência Documental com Proveniência Verificável sobre
Modelo Relacional*.

| Arquivo | O que é |
|---|---|
| `artigo.tex` | Fonte do artigo (modelo SBC). Compila no Overleaf com `sbc-template.sty` e `sbc.bst` na raiz do projeto. |
| `references.bib` | Bibliografia BibTeX |
| `tables/` | Tabelas geradas a partir dos CSV de `results/` |
| `results/` | Resultados brutos, uma linha por pergunta e repetição |
| `anexos/` | Imagens usadas pelo LaTeX (a versão em PDF para a banca está em `../../anexos/`) |
| `Apresentação de TCC ThingIndex Slides.pdf` | Slides da defesa |

Os anexos exigidos pelo colegiado (diagramas, telas, tabelas completas) estão
em PDF na pasta [`anexos/`](../../anexos/README.md) da raiz do repositório,
gerados por `scripts/gerar-anexos.py`.

Os números do texto são conferidos contra os CSV por
`scripts/check-article-numbers.py`; citações, ordem das tabelas e afirmações
verificáveis, pelos demais `scripts/check-article-*`.
