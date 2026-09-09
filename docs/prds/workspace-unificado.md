# Workspace unificado: chat, documentos e upload dentro da Área de trabalho

**Status:** entregue em PR sem merge (2026-09-08), para comparar com a `main`
**Data:** 2026-09-08
**Base:** `docs/prds/workspace-navegacao-e-estado.md` (entregue em WN1–WN6)
**Origem:** pedido do autor: "juntar as páginas Área de trabalho, Dashboard, Chat e Documentos, para enviar documentos, perguntar sobre eles e ver onde estão os textos citados e as conexões, tudo na mesma página". Entregue num PR **sem merge**, para o autor comparar a navegação antiga (`main`) com a nova (branch).

## Goals

(1) Perguntar ao chat de dentro da Área de trabalho e, em cada fonte citada, clicar em "Ver no documento" para o canvas ir ao documento e à página citados (e à seção, quando a fonte vier de um nó do sumário), sem sair da tela. (2) Ver a lista de documentos da coleção no mesmo painel, com enviar, reindexar e apagar, e o canvas refletindo a mudança sem recarregar a página. (3) Menu com uma entrada de trabalho só: Dashboard e Chat deixam de existir como páginas, a biblioteca de Documentos some, e a aba Operações vira a página "Operações". Rotas antigas redirecionam.

Pronto quando: na stack real com o usuário demo, uma pergunta ao chat na coleção de SI devolve fontes e "Ver no documento" leva ao PDF e à página certos; enviar um PDF pelo painel faz um quadro novo aparecer no canvas quando a indexação termina; apagar remove o quadro; `/pt-BR/dashboard` e `/pt-BR/chat?collection=3` abrem a Área de trabalho; E2E com mock cobre cada um desses passos; reversão por asserção nova.

## Requirements

Rodada 1 — chat no workspace (usável sozinha)
- [ ] `ChatSource` (domínio, DTO do stream e `ChatSourceReference` persistido) ganha `document_id` (uint) e `node_id` (string, vazio para hit de página). O `id` continua `doc:page[:node]`. Mensagens antigas sem `document_id` continuam válidas: o botão não aparece.
- [ ] Contrato `lib/api/chat.ts`: `ChatSource` tolerante a `documentId`/`document_id` ausentes. `SourceReference` recebe `onVerNoDocumento?: (source) => void` e mostra o botão "Ver no documento" só quando há `documentId`.
- [ ] `ChatTab` aceita `collectionId` controlado (esconde o seletor de coleção quando controlado), `onVerNoDocumento` e `compacto` (cabeçalho de uma linha, sem ações rápidas). `useChatWorkspace` aceita `collectionId` inicial/controlado; continua gravando `?collection`/`?conversation` na URL (já é agnóstico à rota).
- [ ] Painel de chat à **esquerda** do canvas, 380 px em `lg`, recolhível por botão no cabeçalho da Área de trabalho ("Chat" com ícone; estado local, aberto por padrão em `lg`, fechado abaixo). Abaixo de `lg` o painel fica acima do canvas com altura máxima de 40 vh. A lista de conversas fica dentro do painel, no topo, usando `ChatConversationRail` (hoje sem uso).
- [ ] "Ver no documento" chama `irAte(documentId, page, rectDaSecao)` no `WorkspaceCanvas` e foca o documento (`documentoFocado`). Quando `nodeId` existe e o nó tem `paragraph_start` com box, o rect é o desse parágrafo; senão, topo da página. O canvas expõe isso via `ref` imperativo (`useImperativeHandle` com `irAte`), para a página ligar o chat ao canvas sem levantar o estado inteiro.
- [ ] A Área de trabalho lê `?collection=` na URL para escolher a coleção inicial (hoje escolhe a primeira). Trocar a coleção no seletor atualiza a URL.
- [ ] Testes: unitário do `buildChatSources` (document_id e node_id nos dois tipos de hit); unitário do `SourceReference` (botão só com documentId, chama o callback); E2E mock `workspace-chat.spec.ts`: stream mockado devolve fonte com `document_id` e `page: 3`; clicar em "Ver no documento" muda o quadro para p. 3 (`data-pagina-atual` no modo página) ou centraliza a página 3 (viewport muda); fonte sem `document_id` não mostra o botão.

Rodada 2 — documentos e upload no workspace
- [ ] O painel esquerdo ganha duas abas: "Chat" e "Documentos". A aba Documentos lista `context.documents` (título, status com a mesma cor de badge da biblioteca, páginas quando medidas). Clicar no título chama `focarDocumento`.
- [ ] Menu por documento: "Reindexar" (`retryDocumentIndexing`) e "Apagar" (`deleteDocument`, com `DangerConfirmModal` já existente). Depois de qualquer ação, a página recarrega o contexto (`carregarContexto`) e o canvas recebe o contexto novo pela prop (a `key` por coleção não muda, então posições e estado ficam).
- [ ] Botão "Enviar documento" abre um modal com o `UploadTab` em modo `colecaoFixa={id}` (sem combobox de coleção, categoria opcional, sem guia rápido). Ao fechar o modal com envio feito, a página passa a recarregar o contexto a cada 5 s enquanto houver documento com status diferente de `INDEXED`/`FAILED`, e para quando não houver.
- [ ] Testes: unitário do `UploadTab` com coleção fixa (não renderiza combobox, envia com a coleção); E2E mock `workspace-documentos.spec.ts`: apagar remove o quadro do canvas; upload mockado + contexto mockado com documento `PROCESSING` → após o segundo GET retorna `INDEXED` e o quadro aparece.

Rodada 3 — navegação
- [ ] Menu do `AppShell`: Área de trabalho, Operações (`/documents`), Feature futura, Documentação, Configurações. Somem Dashboard e Chat, e some o bloco especial do chat no menu (rail de conversas) porque a lista vive no painel.
- [ ] Redirecionamentos em `next.config.ts` (`redirects()`): `/:locale/dashboard` → `/:locale/workspace`; `/:locale/chat` → `/:locale/workspace` preservando query (`?collection`, `?conversation`, `?new`). Login e registro passam a redirecionar para `/workspace`; `PROTECTED_PATHS` do `proxy.ts` ganha `/workspace` e perde `/dashboard`.
- [ ] `app/documents/page.tsx` perde a aba biblioteca: somem `DocumentCard`, `DocumentPreviewPanel`, `FilePreview`, `IndexStructureView`, `LibraryControlPanel` e o estado de filtros; a página vira só a aba de operações, com título "Operações". `app/dashboard/page.tsx`, `app/chat/page.tsx` e as versões em `[locale]` são apagados; `AccountStatus` e `DashboardQuickGuide` são apagados se ficarem sem uso.
- [ ] i18n (pt-BR, en, es): chaves novas em `workspace.chat.*`, `workspace.documents.*`, `shell.nav.operations`; chaves órfãs do dashboard removidas só se um grep provar que não são usadas.
- [ ] E2E existentes que abrem `/dashboard`, `/chat` ou a biblioteca (`chat-workspace`, `runtime-visibility`, `audit-explorer`, `document-observability`, `telas-autenticadas`, `dashboard-cost-visibility`) são ajustados para as rotas novas; `chat-workspace.spec.ts` vira o teste do painel dentro do workspace. E2E novo `navegacao-redirects.spec.ts`: as duas rotas antigas redirecionam com query preservada.
- [ ] Real: `tests/e2e-real/percurso-completo.spec.ts` ganha o passo "enviar documento pelo painel e ver o quadro aparecer".

## Non-Goals

- Mudar o motor do chat, o retrieval ou o prompt. Só a forma da fonte muda.
- Chat sem coleção selecionada. Chat em várias coleções.
- Mover Operações para dentro do workspace. Mover Configurações.
- Destacar o trecho exato citado dentro da página (a unidade é página, ou seção quando o hit é de nó). Busca do snippet dentro do texto da página fica para depois.
- Persistir se o painel esquerdo está aberto ou qual aba está ativa.
- Landing page (`/`) e seus links.

## Constraints

- `ChatSourceReference` é jsonb com serializer JSON: campos novos com `omitempty` são compatíveis com linhas antigas. Nenhuma migração.
- `useChatWorkspace` já escreve `?collection`/`?conversation` com `router.replace` no `pathname` atual e dispara `thingindex:chat-sidebar-sync`; o `AppShell` só escuta esse evento na rota `/chat`. Na rodada 1 o `AppShell` não muda; na rodada 3 o bloco especial sai.
- `WorkspaceCanvas` já tem `irAte(documentId, pagina, rect)` e `focarDocumento(id)`; `rectDaPagina` e a âncora de seção via `paragraph_start` já existem. O `ref` imperativo é a menor ponte entre página e canvas.
- `UploadTab` decide a coleção por `defaultCollection` (string) e `CollectionCombobox`; a coleção fixa entra como prop e o combobox some. `useDocumentUpload` não muda.
- `context.documents` (`DocumentSummary`) já traz `status`, então a lista da rodada 2 não precisa do `useDocuments` (que lista todos os documentos do usuário e faz polling próprio).
- `proxy.ts` protege por prefixo e hoje não inclui `/workspace` nem `/chat`; a rodada 3 corrige.
- E2E com mock: os testes existentes de workspace interceptam `**/api/documents/collections/*/context`; o chat intercepta `**/api/chat/stream` com um corpo SSE. O E2E novo combina os dois.
- Não há provedor de LLM no usuário demo da stack real; o E2E real da rodada 1 não pode perguntar ao chat. Fica no mock; registrado em Discoveries.

## Approach

1. Backend: `document_id`/`node_id` em `domain.ChatSource`, `models.ChatSourceReference` e nas duas conversões em `chat_usecase.go`; teste unitário.
2. Frontend contrato + `SourceReference` com botão + `ChatTab`/`useChatWorkspace` controlados.
3. `WorkspaceCanvas` com `forwardRef` e `useImperativeHandle({ irAte, focarDocumento })`; `page.tsx` monta o painel esquerdo, lê `?collection`, liga `onVerNoDocumento` ao ref.
4. Aba Documentos + modal de upload + recarga do contexto com polling curto.
5. Navegação: menu, redirects, proxy, login, remoção da biblioteca e das páginas, i18n, E2E ajustados.
6. PR aberto, sem merge, com prints das duas versões na descrição.

## Open Questions

- Painel à esquerda ou à direita? — *decidido por padrão: esquerda*, porque a sidebar de estrutura já está à direita e o chat é "entrada", a estrutura é "saída".
- Fonte de página inteira ou tentar achar o parágrafo pelo snippet? — *decidido por padrão: página (ou seção via `node_id`)*; snippet fica em Non-Goals.
- O que fazer com o histórico de conversas do `/chat`? — *decidido por padrão: nada muda no banco*; `/chat?conversation=N` redireciona para `/workspace?conversation=N` e o painel abre a conversa.
- Manter `/dashboard` como página inicial magra? — *decidido por padrão: não*; o autor disse que faria sumir.

## Decisões tomadas sem entrevista (autor ausente durante a execução, 2026-09-08)

- Painel esquerdo, 380 px, recolhível, não persistido.
- Alvo da fonte: página; seção quando `node_id` existir.
- Três rodadas, cada uma usável sozinha, no mesmo PR, sem merge.
- Dashboard some; Operações vira página própria em `/documents`.

## Alternativas descartadas

- **Levantar o estado do canvas para a página** (em vez de `ref` imperativo com `irAte`/`focarDocumento`): obrigaria a mover viewport, apresentações e seleção para `page.tsx` e reescrever o `WorkspaceCanvas`. O `ref` é uma ponte de duas funções.
- **Componente de chat novo** para o painel, em vez de `ChatTab` com props: duplicaria composer, stream, sub-queries e rail. `ChatTab` já é autocontido; três props resolvem.
- **Redirecionar no `proxy.ts`** em vez de `redirects()` do `next.config.ts`: o proxy já cuida de locale e auth; misturar redirects de rota nele espalha regra. `redirects()` preserva query por padrão e é declarativo.
- **Manter `/dashboard` como página inicial magra** (status da conta e guia): o autor disse que a faria sumir; status da conta não tem uso fora dela.
- **Listar documentos com `useDocuments`** (todos do usuário, com polling próprio): `context.documents` já traz `status` e é da coleção certa; o polling curto na página cobre o upload.

## Ambiguity Report (grill-me, modo spec, 2026-09-08)

Suposições verificadas no código: `useChatWorkspace` só escolhe coleção quando `selectedCollectionId` é `null` (aceita valor inicial e `setSelectedCollectionId` para sincronizar); `OutlineNode.nodeId` existe e a âncora de seção por `paragraph_start` está em `WorkspaceCanvas.tsx:247`; `ChatSourceReference` é jsonb via serializer JSON; `DangerConfirmModal` tem `isOpen/title/description/confirmLabel/expectedValue/confirmationValue/isPending/onConfirm`; `next.config.ts` não tem `redirects()` ainda; `proxy.ts` protege `/dashboard, /documents, /cost-visibility, /settings` e manda o logado para `/dashboard`.

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (E2E real do chat impossivel sem provedor; fica no mock)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (cinco alternativas nomeadas; direita x esquerda do painel decidido por padrao)
  Assumptions:  0.0   ✓ clear (todas verificadas no codigo)
  ──────────────────────────────
  Aggregate:    0.10  ✓ below threshold (0.2 spec)
```

## Discoveries

- O `DocumentPreviewPanel`, `FilePreview` e `IndexStructureView` NAO saíram da página de Documentos: a aba de triagem abre o documento prioritário por eles ("Open priority document" → aba Index). O PRD listava os três para remoção; só a biblioteca (cartões, painel de controle, agrupamento) saiu. 952 linhas a menos mesmo assim.
- A lista de conversas do menu (renomear, excluir) ficou no `AppShell`, agora pendurada no item Área de trabalho, porque o painel compacto só tem um `select` de conversas e um botão de nova conversa. `ChatConversationRail` continua sem uso, como antes.
- Com o painel aberto o segundo quadro fica fora da tela e as páginas só são renderizadas quando visíveis. Três consequências: (1) "Ver no documento" precisa trocar a página mesmo de documento ainda não medido (`irAte` ganhou esse caso); (2) os E2E que contam grifos nos dois quadros fecham o painel primeiro; (3) os specs antigos do workspace passaram a mockar runtime e conversas, senão as buscas do chat recebem 401 e a página cai no login.
- Playwright dá prioridade à última rota registrada: o mock por id (`/conversations/*`) precisa vir depois do mock da lista (`/conversations**`).
- Depois do stream o hook recarrega a conversa pelo id; o mock tem de devolver as mensagens persistidas (snake_case), senão a resposta some da tela.
- O polling pós-upload não pode decidir a parada num efeito: ele rodaria com o contexto velho (sem o documento novo) e pararia antes da primeira recarga. A decisão ficou dentro do tick, lendo o contexto por ref.
- `useSearchParams` na página exige `Suspense` na pré-renderização; a página foi envolvida.
- O E2E real do chat não é possível: o usuário demo não tem provedor de LLM. O painel e a fonte clicável estão cobertos só com mock; upload, apagar e estado estão cobertos na stack real.
- Entregue em 6 issues (#48 a #53) mais o fechamento (#54). Suítes: Vitest 225, Playwright mock 36, Playwright real 4, `go test ./...` ok.
