# Artefatos acadêmicos — TCC ThingIndex

Autor: Charles Eduardo Mello Guimarães — IFPR Campus Palmas, Sistemas de Informação.

Este repositório é **privado**, então os artefatos de pesquisa ficam versionados
aqui junto do código a que se referem.

## Backup principal

| Arquivo | O que é |
|---|---|
| **`thingindex-artigo.pdf`** | **Versão compilada de referência do artigo** — 23 páginas, A4, saída do LaTeX. É este o backup principal. |
| `article-overleaf.md` | Fonte em Markdown usada no Overleaf |
| `references.bib` | Bibliografia BibTeX |
| `Apresentação de TCC ThingIndex Slides.pdf` | Slides da defesa |
| `anexos/` | Figuras e diagramas (banco, arquitetura) |

Título: *ThingIndex: Plataforma Open Source para Inteligência Documental com
Baseline Relacional*.

## Por que isto está versionado

Até agosto de 2026, a linha `docs/` no `.gitignore` da raiz excluía esta pasta
inteira. O resultado é que **o artigo, a bibliografia e os slides não tinham
backup nenhum** — existiam apenas no disco local, sem histórico e sem cópia.
Uma falha de disco custaria o trabalho inteiro.

A regra foi removida. Ao atualizar o artigo, commite o PDF junto: é ele que
garante que existe sempre uma versão compilada recuperável, independente do
estado do Overleaf.

## Material relacionado

- `../analise-para-correcao/` — análise do referencial teórico por seção
- `../presentation/apresentacao.md` — roteiro da apresentação
- `../estado-do-projeto.md` — estado de engenharia, o que falta e o que não mexer
- `../../api/internal/usecase/testdata/indexing-benchmark/corpus.json` — corpus
  controlado que sustenta os números de benchmark citados no artigo
