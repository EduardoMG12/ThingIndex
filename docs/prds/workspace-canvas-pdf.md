# Workspace de proveniência como canvas (estilo Figma/Miro) com o PDF renderizado

**Status:** entregue (2026-09-08)
**Data:** 2026-09-08
**Origem:** pedido do autor ("como se fosse um figma mesmo, um miro, e no meio renderizasse o pdf"), sobre a aba Workspace entregue na spec 001 (P4)

## Goals

A aba Workspace deixa de ser um painel plano de cartões e vira um canvas infinito
com pan e zoom. O documento focado aparece como as páginas reais do PDF, com cada
parágrafo indexado grifado na posição exata que a indexação gravou (bounding box).
As conexões entre trechos de documentos diferentes são desenhadas como linhas do
grifo de origem até o destino, com o score visível e os termos casados no hover.

Está pronto quando: (1) o E2E com mock abre a coleção, foca um PDF, vê os grifos
alinhados com o texto, pede conexões e vê as linhas com score; (2) o E2E na stack
real, com `court-order.pdf` e `supplier-master-order-form.pdf`, conta tantos grifos
quanto parágrafos com `box` no contexto; (3) nenhuma tabela nova no banco.

## Requirements

Canvas
- [ ] Pan arrastando o fundo ou com a roda (convenção Figma/Miro); zoom com `Ctrl`/`Cmd`+roda e pinch, ao redor do cursor; limites 0,1x a 4x; botões "+", "−" e "ajustar à tela" (enquadra todos os quadros); `Esc` limpa a seleção.
- [ ] Cada documento é um quadro no canvas. Posição inicial em grade determinística (mesma coleção, mesmo arranjo). Arrasto do quadro e persistência em `localStorage` continuam como hoje.
- [ ] Clicar num quadro foca o documento (sidebar passa a mostrar o sumário dele). O foco vai para a URL (`?doc=<id>`) para a tela ser compartilhável e reproduzível em demonstração. Foco é só seleção: não muda o que é renderizado.

Quadros de PDF (todos, não só o focado)
- [ ] Cada PDF é um quadro com as páginas empilhadas verticalmente em tamanho real. As dimensões das páginas vêm do `pdfjs-dist` ao abrir o arquivo (metadados, barato); o conteúdo de cada página só é renderizado quando ela entra no viewport.
- [ ] O arquivo vem de `/api/documents/{id}/preview` (já serve inline com o cookie de sessão). Ele é baixado uma vez por documento e só quando o quadro entra no viewport pela primeira vez; antes disso o quadro mostra título, contagem de trechos e um esqueleto do tamanho estimado.
- [ ] Orçamento de renderização, para não pesar no navegador: no máximo 12 páginas com `<canvas>` vivo ao mesmo tempo, em todos os quadros somados; página que sai do viewport libera o canvas e volta a placeholder (mantém só o tamanho). Escala de renderização = `min(zoom, 2) × min(devicePixelRatio, 2)`. Ao mudar o zoom, escala por CSS na hora e re-renderiza com nitidez 150 ms depois de o zoom parar. Abaixo de zoom 0,35x nenhuma página é renderizada: o quadro mostra só a miniatura da primeira página (escala 0,25, renderizada uma vez e cacheada por documento).
- [ ] Carregamento e renderização usam um único worker do pdf.js compartilhado; documentos são abertos sob demanda e fechados (`destroy()`) quando saem do viewport por mais de 30 s.
- [ ] Cada `ParagraphNode` com `metadata.box` vira um retângulo translúcido sobre a página, convertido de pontos PDF para pixels com `page_width`/`page_height` do próprio box. Clicar seleciona o trecho; o selecionado ganha cor de destaque.
- [ ] Parágrafos sem `box` (OCR via tesseract, HTML, XLSX, PDFs indexados antes de hoje) não são desenhados sobre a página; aparecem na sidebar marcados "sem posição".
- [ ] PDF focado em que nenhum parágrafo tem `box` mostra um aviso no topo do quadro com o botão "Reindexar", que chama a rota de retry existente. Nada é disparado sem clique: com LLM configurado, reindexar custa dinheiro.
- [ ] Marcador de seção na margem esquerda da página em que o `OutlineNode` começa (`start_page`), com o título do nó.
- [ ] Cabeçalho discreto por página com número e origem do texto (`pdftotext`, `ocr`, `ocr_unlimited`, `pdf_text`), tirada do `metadata.source` dos parágrafos daquela página.

Documentos que não são PDF
- [ ] HTML e XLSX: cartão com título e contagem de trechos, sem páginas nem miniatura. Os trechos ficam só na sidebar.

Conexões
- [ ] Com um parágrafo selecionado, "Ver conexões" chama a API de conexões com o `nodeId` do nó de sumário cujo `slug` é igual ao `section_key` do parágrafo. Nada é persistido (princípio II da constituição: aresta derivada, nunca armazenada).
- [ ] Cada conexão é uma linha SVG em coordenadas do canvas, do retângulo de origem até o destino: se o documento destino está focado, até o marcador de seção do `nodeId` destino, e os parágrafos dessa seção (mesmo `section_key`) ficam grifados em cor secundária; senão, até a borda do quadro dele.
- [ ] Score impresso na linha; termos casados em tooltip no hover; espessura ou cor por faixa de score. Lista vazia mostra "nenhuma conexão acima do limiar", como hoje.
- [ ] Clicar numa conexão na sidebar centraliza e dá zoom no destino.

Sidebar (mantida)
- [ ] Sumário hierárquico do documento focado; clicar numa seção rola o quadro até a página dela.
- [ ] Detalhe do trecho selecionado: conteúdo, seção, página, origem.
- [ ] Lista de conexões com score e termos, sincronizada com as linhas.

Acessibilidade e i18n
- [ ] Grifos e marcadores são botões com `aria-label` ("Parágrafo 3, página 2, seção 1 RELATORIO"). Zoom e "ajustar à tela" acessíveis por teclado.
- [ ] Textos novos em PT-BR, EN e ES no dicionário existente.

Testes
- [ ] Unitário puro para a geometria: box em pontos → pixels; zoom ao redor do cursor; ponto de saída/entrada das linhas. Sem DOM.
- [ ] E2E com mock: fixture PDF pequeno servido pela rota `/api/documents/{id}/preview` mockada, contexto com `box` e `start_page`. Cobre foco, grifo, marcador, conexão desenhada e estado vazio. "Alinhado" é verificável: o retângulo do grifo tem `left/top/width/height` iguais a `box × (tamanho do elemento da página / page_width|page_height)`, com tolerância de 1 px, medido por `boundingBox()`.
- [ ] E2E na stack real (`tests/e2e-real`): conta grifos = parágrafos com `box`.
- [ ] Os 3 testes atuais de `workspace.spec.ts` e a varredura de `telas-autenticadas` continuam verdes (adaptados onde o layout mudou).
- [ ] Teste de orçamento: com 3 PDFs de 30 páginas mockados, nunca há mais de 12 `<canvas>` de página no DOM ao rolar o canvas de ponta a ponta.

Corpus de demonstração: uma faculdade
- [ ] Script `scripts/seed-demo-faculdade.sh` que registra um usuário de demonstração e envia, pela API, documentos públicos reais de uma instituição de ensino (editais, regimento, calendário acadêmico, resoluções), organizados em categorias e coleções, por exemplo: categoria "Editais" com coleção "Vestibular 2026"; categoria "Normas" com coleção "Regimento e resoluções"; categoria "Calendário" com coleção "Ano letivo 2026".
- [ ] Os PDFs ficam versionados em `examples/faculdade/` com um `FONTES.md` listando URL de origem, data de captura e licença/uso público de cada um. Só documentos publicados abertamente pela instituição.
- [ ] O script espera todos chegarem a `INDEXED` e imprime, por documento: páginas, parágrafos, parágrafos com `box`, e falha se algum documento terminar `FAILED` ou sem nenhum `box` (um PDF digitalizado sem OCR de caixa é aceito, mas precisa ser declarado no `FONTES.md`).
- [ ] O workspace aberto nessa coleção é o cenário da demonstração e do E2E real: pelo menos uma conexão com score ≥ 2 entre dois documentos diferentes deve existir e ser desenhada.

## Non-Goals

- Minimapa, seleção múltipla, alinhamento automático, agrupamento.
- Anotação, comentário ou edição de qualquer coisa sobre o PDF.
- Seleção de texto dentro do PDF (camada de texto do pdf.js).
- Manter todas as páginas de todos os PDFs renderizadas ao mesmo tempo (o orçamento de 12 páginas vivas é regra, não sugestão).
- Persistir arestas, posições no servidor ou qualquer tabela nova.
- Suporte a toque/mobile além do que os pointer events derem de graça. Desktop primeiro.
- Calcular conexões de todos os nós ao abrir a coleção.
- Extrair bounding box para o caminho tesseract (segue como limitação declarada no artigo).

## Constraints

- **Bounding box só existe em dois caminhos:** `pdftotext -bbox-layout` e Unlimited-OCR. O tesseract e o fallback puro em Go não emitem. Até hoje (2026-09-08) a imagem dev do indexador não tinha poppler, então 0 de 12.665 parágrafos do banco local têm `box`; corrigido em `f02b2a5`. Documentos antigos precisam de reindexação (rota de retry existente) para ganhar grifo.
- **Formato do box:** `{x_min, x_max, y_min, y_max, page_width, page_height}` em pontos PDF, origem no topo esquerdo, `/Rotate` e `CropBox` já aplicados. A conversão é linear: `px = x / page_width * larguraRenderizada`.
- **Mapeamento parágrafo → nó de sumário:** `paragraph.metadata.section_key === outline.slug`. A API de conexões recebe o `node_id` do sumário (`node-001-1-relatorio`), não o id do parágrafo.
- **`start_page` do sumário** existe no backend e no JSON (`start_page`), mas o contrato do frontend (`OutlineNode`) não o normaliza. Precisa entrar.
- **Renderização com `pdfjs-dist` no navegador** (decisão do autor, 2026-09-08). Alternativas consideradas: rasterizar no servidor com `pdftoppm` e servir PNG por rota nova (zero dependência nova, mas CPU na API a cada pedido e um endpoint Go a mais); `<embed>`/`<iframe>` (descartado: não permite sobrepor grifos alinhados). O custo aceito é ~300 kB gzip e a configuração do worker no Next via `import.meta.url`; o orçamento de páginas vivas é o que mantém o navegador leve.
- **Score de conexão é inteiro:** contagem de termos significativos da origem encontrados no destino (`retrievalScore`), mínimo padrão 2. Não é probabilidade. A espessura da linha cresce com a contagem (2, 3, 4+) e o rótulo diz "3 termos", nunca "0,75".
- **Reindexação usa a rota de retry existente** (`POST /documents/:id/retry`), que aceita documento já `INDEXED` (só volta o status para fila). Não há precondição de status no usecase.
- **Pan e zoom sem biblioteca:** CSS `transform` num contêiner "mundo", como o canvas atual já faz para arrasto. React Flow fica descartado pelo mesmo motivo registrado na spec 001; revisitar só se minimapa ou layout automático entrarem em escopo.
- **Arestas derivadas, nunca armazenadas** (constituição, princípio II). A prova de não-persistência do PR #20 (`COUNT(*)` antes e depois) continua valendo e deve seguir verde.
- **Lint:** `react-hooks/set-state-in-effect`, `refs` e `immutability` estão como aviso desde hoje. Código novo não deve adicionar ocorrências.
- **Preview via BFF:** `/api/documents/{id}/preview` já faz proxy autenticado para `GET /documents/:id/content` com `Content-Disposition: inline`. Não precisa de rota nova.

## Approach

1. `lib/workspace/geometry.ts`: funções puras (box → px, zoom ao redor do ponto, âncoras de linha). Testadas primeiro.
2. `components/workspace/useViewport.ts`: estado `{x, y, escala}` com pan, zoom por roda, ajustar à tela. Aplica `transform: translate() scale()` no contêiner mundo.
3. `components/workspace/PdfFrame.tsx`: abre o PDF com `pdfjs-dist` a partir da rota de preview quando o quadro entra no viewport, cria placeholders com o tamanho real de cada página, renderiza em `<canvas>` só as páginas visíveis dentro do orçamento global (`lib/workspace/render-budget.ts`, puro e testado), sobrepõe a camada de grifos e marcadores em `<div>` absolutos. Miniatura = página 1 em escala 0,25, cacheada por `documentId`.
4. `components/workspace/ConnectionLayer.tsx`: `<svg>` do tamanho do mundo com uma `<path>` por conexão, rótulo de score e `<title>` com os termos.
5. `WorkspaceCanvas.tsx` passa a compor as peças acima e mantém a sidebar. `page.tsx` continua carregando o contexto; ganha leitura/escrita de `?doc=`.
6. Contrato: `OutlineNode.startPage` normalizado de `start_page`; `ParagraphNode.metadata.box` tipado numa função `readBox()` que devolve `null` quando ausente ou malformado.
7. E2E: fixture `tests/e2e/fixtures/uma-pagina.pdf` (gerada uma vez, versionada, < 5 KB); mocks de contexto ganham `box` e `start_page`.

8. `scripts/seed-demo-faculdade.sh` + `examples/faculdade/`: corpus real, capturado e documentado, enviado pela API. Usado pelo E2E real e pela demonstração.

Ordem de execução em uma entrega só, como pedido, mas commits atômicos: corpus da faculdade (valida coleções e categorias antes de qualquer UI) → geometria e orçamento → viewport → PdfFrame → grifos e marcadores → conexões → sidebar e URL → E2E.

## Open Questions

- **O orçamento de 12 páginas vivas e o piso de zoom 0,35x** são valores iniciais. Medir com o corpus da faculdade (3 a 6 PDFs, alguns com dezenas de páginas) e ajustar antes de fechar. — *needs exploration*, não bloqueia o início.
- **Escala real do Unlimited-OCR** (`THINGINDEX_OCR_BOX_SCALE`) nunca foi validada por falta de GPU; grifo de página OCR pode sair deslocado. — *blocked on external* (GPU). Não bloqueia esta entrega.

## Decisões tomadas na entrevista (2026-09-08)

- Canvas com todos os documentos; o focado em tamanho real, os demais em miniatura.
- Sobre o PDF: grifo por parágrafo, marcador de seção na margem e cabeçalho de página com origem.
- Conexões como linhas SVG com score; destino focado recebe a linha no marcador de seção e grifo secundário nos parágrafos da seção.
- PDFs antigos sem box: aviso com botão "Reindexar", nunca automático.
- Renderização com pdf.js no navegador, com carregamento por página visível e orçamento de páginas vivas, porque o autor quer vários PDFs no canvas sem pesar.
- Roda = pan; Ctrl/Cmd+roda e pinch = zoom.
- Corpus de demonstração real de uma faculdade, capturado de fontes públicas, versionado e semeado por script, como cenário de validação de coleções e categorias.
- Entrega única, com commits atômicos por peça.

## Discoveries

- 2026-09-08: imagem dev do indexador sem poppler; nenhum box no banco local. Corrigido antes deste PRD (`f02b2a5`).
- O build padrão do pdf.js 6 usa `Map.prototype.getOrInsertComputed`; o Chromium 144 não tem e a página ficava branca com TypeError. O projeto usa o build `legacy`, que traz os polyfills.
- Abrir o PDF e medir as páginas no mesmo efeito não funciona: o `setDocumento` reexecuta o efeito e o cleanup cancela a medição em voo. São dois efeitos.
- "Ajustar à tela" pela altura jogava o zoom em 10% com quadros de 19 mil px. Ajusta pela largura, topo alinhado.
- O `section_key` do parágrafo só cobre 175 das 321 seções do Edital 56; a âncora do marcador usa `metadata.paragraph_start` do nó de sumário, que a indexação já grava. Marcadores só até o nível 2 e dentro da margem esquerda; sem âncora, sem marcador.
- Uma seção do Edital 56 devolve 109 conexões acima do limiar. O canvas desenha as 12 mais fortes; a sidebar lista todas.
- Quadros só abrem quando entram no viewport. Um teste que põe o PDF em terceiro lugar numa tela de 1280 px nunca o vê abrir, e isso é o comportamento certo.
- Entregue em 8 issues (#26 a #33), commits `8b53910` a WS8. Corpus da faculdade: 6/6 documentos, 6.752 parágrafos com box, conexões em todas as collections.

## Ambiguity Report (grill-me, modo spec, 2026-09-08)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (tolerância de alinhamento e orçamento de canvases definidos; números iniciais a medir)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (renderização e pan/zoom comparados; roteamento das linhas não)
  Assumptions:  0.25  ✓ clear (worker do pdf.js no Next 16 ainda não exercitado)
  ──────────────────────────────
  Aggregate:    0.15  ✓ below threshold (0.2 spec)
```

Push lightly on: worker do pdf.js no Next 16 (validar na primeira peça do PdfFrame), medição do orçamento com o corpus da faculdade.
