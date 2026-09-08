# Workspace: grifos coloridos com legenda, folhear página a página e ir até a conexão

**Status:** entregue (2026-09-08)
**Data:** 2026-09-08
**Base:** `docs/prds/workspace-canvas-pdf.md` (entregue em WS1–WS8, commits `8b53910`…`a3220f9`)
**Origem:** pedido do autor depois de testar o canvas com o corpus da faculdade

## Goals

Sobre o canvas que já existe, o usuário passa a (1) escolher, por documento, entre
ver as folhas empilhadas ou uma página por vez com avançar e voltar; (2) marcar
parágrafos com uma de cinco cores cujo significado ele define numa legenda da
coleção, e essas marcas sobrevivem a recarregar a página, trocar de navegador e
de máquina; (3) a partir de um parágrafo marcado, ver as conexões desenhadas até
a parte correspondente do outro PDF e, quando o destino está em outra página,
ir até ela com um clique.

Está pronto quando: o E2E com mock cobre os três fluxos; o E2E na stack real marca
um parágrafo, recarrega e a marca continua; a legenda editada aparece nos
rótulos de acessibilidade; e o teste de isolamento prova que um usuário não vê
as marcas de outro.

## Requirements

Apresentação por documento
- [ ] Cabeçalho de cada quadro de PDF ganha um seletor com dois estados: "folhas" (todas as páginas empilhadas, como hoje) e "uma página" (só a página atual, com ◀ ▶ e "p. 3 de 24"). Padrão: folhas.
- [ ] Em "uma página", o quadro tem a altura de uma página; ◀ ▶ e as teclas PageUp/PageDown com o quadro focado trocam a página; os grifos, marcadores e cabeçalho da página atual continuam desenhados; o orçamento de canvases conta a página atual como qualquer outra.
- [ ] O modo e a página atual de cada documento ficam no `localStorage` junto das posições (preferência de tela, não dado do usuário); a URL não os carrega.
- [ ] Ir até uma conexão ou seção cujo destino está em outra página: em "folhas", centraliza a página (como hoje); em "uma página", troca a página atual daquele quadro e centraliza.

Grifos coloridos
- [ ] Paleta fixa de cinco cores com chaves estáveis: `amarelo`, `verde`, `azul`, `vermelho`, `roxo`. Contraste do texto sobre cada cor conferido (AA sobre branco).
- [ ] Ao selecionar um parágrafo (clique no grifo automático), aparece uma barra flutuante logo acima dele, dentro da página, com os cinco chips de cor, "sem cor" e "Ver conexões". Chip da cor atual aparece marcado. `Esc` fecha.
- [ ] Parágrafo com cor fica com fundo translúcido dessa cor e borda sólida; o grifo automático sem cor continua discreto como hoje. Selecionado ganha anel por cima da cor.
- [ ] A marca é persistida no backend por usuário: `PUT /documents/collections/:id/highlights/:paragraphNodeId` com `{color}` (upsert), `DELETE` para remover, `GET /documents/collections/:id/highlights` devolve todas da coleção. Escopo transitivo por `user_id` (coleção do usuário, parágrafo de documento da coleção).
- [ ] Tabela nova `paragraph_highlights` (`id, user_id, collection_id, document_id, paragraph_node_id, color, created_at, updated_at`), única em `(user_id, paragraph_node_id)`, FK para `paragraph_nodes` com `ON DELETE CASCADE`. Entra em `ManagedModels()` na posição certa (depois de `paragraph_nodes`).
- [ ] Salvar é otimista: a cor aparece na hora, e se a API falhar volta e mostra o erro na sidebar.

Legenda
- [ ] Painel fixo no canto superior direito do canvas, recolhível, com as cinco cores e um campo de texto por cor (rótulo). Rótulo vazio mostra o nome da cor.
- [ ] Rótulos persistidos por coleção em `collections.metadata.workspace_legend` (`{amarelo: "prazo", ...}`) via `PUT /documents/collections/:id/legend`; `GET .../legend` devolve. Sem tabela nova: a coleção já é do usuário e já tem `metadata` jsonb.
- [ ] Os rótulos entram no `aria-label` dos grifos coloridos ("Parágrafo 12, página 3, seção 2 PRAZOS, grifo amarelo: prazo") e no `title` dos chips.
- [ ] Sidebar ganha a contagem de grifos por cor do documento focado, usando os rótulos, e cada linha filtra a lista de parágrafos marcados; clicar num item vai até o parágrafo.

Conexões a partir do grifo
- [ ] "Ver conexões" na barra flutuante usa o mesmo caminho de hoje (nó de sumário da seção do parágrafo). A linha sai do grifo colorido e chega ao primeiro parágrafo da seção destino (âncora de hoje).
- [ ] Cada conexão, na sidebar e no rótulo da linha, mostra "p. N" do destino e um botão "Ir para p. N" quando o destino está em página diferente da visível daquele quadro (sempre, em "uma página"; em "folhas", quando a página não está no viewport). O botão aplica a regra de "ir até" acima.
- [ ] Ao pedir conexões de um parágrafo colorido, os parágrafos da seção destino ganham o grifo secundário com a MESMA cor de origem, mais claro, para o olho seguir a cor.

Testes
- [ ] Backend: teste de integração do repositório com isolamento (usuário B não lista nem apaga marca do usuário A; marca em parágrafo de coleção alheia é 404), upsert idempotente, cascade ao apagar documento; handler com cor inválida → 400.
- [ ] Frontend unit: barra flutuante (chips, cor atual, sem cor), legenda (rótulo vazio → nome da cor, rótulo no aria-label), modo "uma página" (só a página atual no DOM, ◀ ▶ nos limites), regra "ir até" nos dois modos.
- [ ] E2E mock: marcar cor → PUT enviado com a cor; recarregar com GET devolvendo a marca → grifo colorido; editar legenda → PUT e aria-label atualizado; "uma página" → ◀ ▶ e "p. 2 de 30"; conexão com "Ir para p. N" troca a página do quadro destino.
- [ ] E2E real (`tests/e2e-real`): marcar um parágrafo do court-order com amarelo, recarregar, marca continua; segundo usuário não a vê.
- [ ] Poder de detecção provado por reversão em cada asserção nova, registrado no commit.

## Non-Goals

- Seleção de texto livre dentro do PDF (camada de texto do pdf.js). A unidade é o parágrafo.
- Cores livres ou mais de cinco. Notas de texto por grifo. Compartilhar marcas entre usuários.
- Persistir modo de apresentação ou página atual no servidor ou na URL.
- Persistir conexões, mesmo as pedidas a partir de um grifo (princípio II).
- Exportar grifos; sincronizar com o chat.

## Constraints

- **Anotação não é recuperação.** A tabela nova guarda escolha do usuário, não deriva nada; a prova de não-persistência de arestas do PR #20 continua valendo e não deve ser tocada.
- **Escopo por usuário é transitivo e precisa de teste** (constituição, Restrições Técnicas): a marca referencia `paragraph_nodes`, cuja coleção pertence a um usuário. O repositório valida a cadeia parágrafo → documento → coleção → `user_id` antes de gravar.
- **Migration é `AutoMigrate` com lista única** em `api/db/models.go` (`ManagedModels()`); a ordem importa por causa das FKs. Testes de integração usam a mesma lista.
- **Padrão a seguir**: `ChatConversation` (modelo com `user_id` + `collection_id`), `domain.ChatConversationRepository` (interface no domínio), `handlers/chat.go` (`c.GetUint("user_id")`, `response.Error` com `i18n.T`), rotas em `routes.go` no grupo `documents` (`/documents/collections/:id/...`), BFF em `frontend/src/app/api/documents/collections/[id]/...`, cliente em `frontend/src/lib/api/collections.ts` com `fetchWithAuthRedirect`.
- **Mensagens de API em PT-BR e EN** (`pkg/i18n/messages.go`) para os erros novos (cor inválida, parágrafo fora da coleção).
- **`collections.metadata`** é `jsonb` e já é lido pelo contexto (`CollectionSummary.metadata`); a legenda cabe ali sem migração. Escrever exige um método de repositório que atualize só essa chave, sem apagar outras.
- **Modo "uma página" muda a geometria do quadro**: `alturaDoQuadro` e `topoDaPagina` recebem o modo; o orçamento de páginas vivas continua global.
- **Lint**: as três regras do React Compiler estão como aviso; código novo não adiciona ocorrência.
- **PPC de 213 páginas** é o caso de estresse de "uma página": trocar de página não pode re-medir nem reabrir o documento.

## Approach

1. Backend, uma fatia: `models.ParagraphHighlight`, `ManagedModels()`, `domain.ParagraphHighlightRepository` + gorm, usecase com validação de cor e de escopo, handler + DTO, rotas, i18n; legenda como método `UpdateCollectionMetadataKey` no repositório de coleção + rota. Testes de integração no repositório e unitário no usecase.
2. BFF: rotas `documents/collections/[id]/highlights/route.ts`, `.../highlights/[paragraphNodeId]/route.ts`, `.../legend/route.ts`, seguindo `content-proxy`/`proxyJsonWithAuthCleanup`.
3. Cliente e contrato: `fetchHighlights`, `putHighlight`, `deleteHighlight`, `fetchLegend`, `putLegend` em `lib/api/collections.ts`; tipos `HighlightColor` (união literal das cinco chaves) e `Legend`.
4. `lib/workspace/highlights.ts` (puro): paleta com classes Tailwind, `rotuloDaCor(legenda, cor)`, `contagemPorCor`, regra `irAte(modo, paginaAtual, paginaVisivel, destino)`.
5. `PdfFrame`: prop `modo` + `paginaAtual` + `onMudarPagina`; em "uma página" renderiza só uma; controles no cabeçalho.
6. `PageOverlay`: cor do grifo, barra flutuante (`HighlightToolbar.tsx`) posicionada acima do parágrafo selecionado.
7. `LegendPanel.tsx` no canvas; sidebar com contagem por cor.
8. `WorkspaceCanvas`: estado de marcas (carregado com o contexto), otimista com rollback, `irAte` unificando conexão e seção; secundários com a cor de origem.
9. i18n nos três idiomas; E2E mock e real; reversões.

Ordem: backend → BFF e cliente → lib pura → folhear → cores e barra → legenda e sidebar → conexões com "ir para p. N" → E2E.

## Open Questions

- **Posição da barra flutuante quando o parágrafo está no topo da página** (não cabe acima): abaixo do parágrafo. — *decidido por padrão*.
- **Limite de marcas por coleção**: nenhum; o PPC tem 4.382 parágrafos e o GET devolve todas de uma vez. Se passar de alguns milhares, paginar. — *needs exploration* após medir com o corpus.
- **Escala do Unlimited-OCR** segue sem validação (GPU). — *blocked on external*, não bloqueia.

## Decisões tomadas na entrevista (2026-09-08)

- Marcas e legenda no backend, por usuário; legenda por coleção.
- Unidade da marca: o parágrafo (clique no grifo automático).
- Cinco cores fixas com rótulo editável por coleção.
- Modo de apresentação por documento, no cabeçalho do quadro.

## Discoveries

- `createForwardHeaders` devolve um `Headers`; espalhar com `{...}` perde o `Authorization` e o PUT via BFF dá 401. Usar `new Headers(...)`.
- As classes de cor moram em `lib/workspace/highlights.ts` e o Tailwind só varria `app/` e `components/`: chips e grifos coloridos saíam invisíveis no purge. `src/lib` entrou no `content`.
- As buscas de marcas e legenda não podem redirecionar em 401: no E2E com token falso a página inteira ia para o login. São dados secundários; viram erro na sidebar.
- No jsdom o `ResizeObserver` nunca dispara, o viewport fica 0×0 e nenhum quadro "entra na tela". Os testes do canvas informam um tamanho real.
- A legenda no canto superior direito cobria o cabeçalho do último quadro; foi para o inferior esquerdo.
- Uma seção de edital devolve 30 a 100 conexões: em "uma página" cada uma ganha "Ir para p. N", e o botão só faz sentido com a página do destino ao lado.
- Entregue em 8 issues (#34 a #41). Stack real: marca amarela sobrevive ao reload, usuário B recebe 404 na coleção de A; Edital 57 folheado vai para a p. 18 ao clicar em "Ir para p. 18" a partir de um grifo verde do Edital 56, com 13 secundários verdes na página.

## Ambiguity Report (grill-me, modo spec, 2026-09-08)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (limite de marcas por coleção fica para medir)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (backend x localStorage, tabela x metadata decididos; barra flutuante x popover na sidebar não comparados)
  Assumptions:  0.25  ✓ clear (escrita parcial em collections.metadata ainda não tem método no repositório)
  ──────────────────────────────
  Aggregate:    0.15  ✓ below threshold (0.2 spec)
```

Push lightly on: método de atualização parcial de `collections.metadata`; medir o GET de marcas com o PPC.
