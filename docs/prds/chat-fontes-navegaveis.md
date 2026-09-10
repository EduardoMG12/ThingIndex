# Chat: resposta direta, fontes numeradas e clique na fonte grifando o trecho no PDF

**Status:** em execução (2026-09-09), branch `feat/conexoes-por-token` (PR #72, sem merge)
**Data:** 2026-09-09
**Origem:** o autor, com duas fotos do chat: a resposta lista três documentos com valores parecidos antes de responder; os nove cards de fonte são iguais e o trecho de uma linha de tabela do PPC aparece como "coluna 1: ...; coluna 12: ..."; e "clicar na referência deveria focar o PDF na página, grifando a fonte, e deixar passar para a próxima fonte".

## Goals

(1) A resposta começa pela resposta: o prompt pede a frase direta primeiro e as citações depois; medido no gabarito (1 repetição) para garantir que o acerto não cai. (2) Cada fonte do chat carrega, além de documento e página, o **parágrafo** mais próximo da pergunta naquela página (`paragraph_id`), escolhido por cobertura de termos; linha de tabela continua carregando a tabela. (3) Clicar numa fonte foca o documento, centraliza a página e **seleciona o parágrafo** (grifo âmbar que já existe) ou realça a caixa da tabela; as fontes são numeradas na ordem do prompt e a mensagem tem setas "fonte anterior / próxima" que percorrem a lista chamando o mesmo foco. (4) Linha de tabela sem cabeçalho real não mostra "coluna N:" no trecho nem no prompt.

Pronto quando: unitário de `buildChatSources` prova que a fonte de página traz o parágrafo com mais termos da pergunta; unitário de `RenderTableRow` prova que cabeçalho sintético some; Vitest do card prova numeração, fonte ativa e setas; E2E com mock do workspace clica numa fonte e vê o grifo selecionado (`aria-pressed`) na página certa.

## Requirements

- [ ] `domain.ChatSource.ParagraphID` (`paragraphId`), `models.ChatSourceReference.ParagraphID` (JSONB, sem migração), `dto.ChatSourceResponse.ParagraphID` (`paragraph_id`).
- [ ] `buildChatSources`: para hit de página e de seção, `paragraph_id` = parágrafo daquela página com maior `queryTermCoverage` sobre os termos significativos de `retrieval.Query`; empate, o primeiro; zero cobertura, nenhum.
- [ ] `RenderTableRow`: cabeçalho que começa com "coluna " é omitido.
- [ ] Prompt de sistema: "Start with the direct answer in one sentence; then give details and cite the document and page of each fact."
- [ ] Frontend: `ChatSource.paragraphId`; `WorkspaceCanvasHandle.irAte(documentId, page, nodeId?, paragraphId?)` seleciona o parágrafo quando existe no contexto; `SourceReference` numerada e com estado ativo; `ChatMessageCard` com índice da fonte ativa e botões anterior/próxima (`data-testid="fonte-anterior"`, `"fonte-proxima"`), que chamam `onVerNoDocumento` e marcam a ativa.
- [ ] Medição: `cmd/generation-eval -arms T -repeats 1` antes/depois do prompt novo; registrar.

## Non-Goals

- Grifar por coordenadas de palavra dentro do parágrafo (o grifo é o parágrafo inteiro).
- Persistir a fonte ativa; estado só na tela.

## Constraints

- `CollectionContext` já carrega `ParagraphNodes` com página e `box`; o canvas já seleciona parágrafo por `selecionarParagrafo(no, secao)` e centraliza por `rectDoParagrafo`.
- `queryTermCoverage`/`significantQueryTokens` já existem em `document_usecase.go`.
- As fontes de mensagens antigas não têm `paragraph_id`: o clique cai na página, como hoje.

## Ambiguity Report (grill-me, modo spec, 2026-09-09)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.0   ✓ clear
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (grifo por palavra adiado; parágrafo já tem caixa)
  Assumptions:  0.0   ✓ clear (verificado no código)
  ──────────────────────────────
  Aggregate:    0.05  ✓ below threshold (0.2 spec)
```

## Discoveries

