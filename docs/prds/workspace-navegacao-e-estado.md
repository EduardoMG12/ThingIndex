# Workspace: navegar entre conexões a partir do grifo, estado de visualização no servidor e marcadores sob controle

**Status:** rascunho
**Data:** 2026-09-08
**Base:** `docs/prds/workspace-anotacoes-e-folhear.md` (entregue em WA1–WA8)
**Origem:** pedido do autor depois de usar o corpus da faculdade

## Goals

(1) Clicar num parágrafo colorido já busca as conexões e abre um navegador
"conexão 1 de N" com anterior e próxima; cada passo leva ao trecho destino no
outro documento (troca a página se estiver folheado, centraliza, mostra título,
página e texto). (2) O estado de visualização de cada coleção (zoom, posição do
canvas, posição dos quadros, modo e página de cada PDF, marcadores visíveis ou
não) é salvo no backend por usuário e restaurado ao abrir, em qualquer máquina.
(3) Os marcadores de seção têm um botão para esconder, e o indexador não cria
seção a partir de linha de calendário (feito em `595ebc4`).

Pronto quando: E2E com mock cobre o navegador de conexões e o estado restaurado;
E2E real folheia um PDF até a página 4, recarrega em outro contexto de navegador
com o mesmo usuário e reabre na página 4; teste de isolamento do estado.

## Requirements

Navegador de conexões
- [ ] Clicar num grifo que já tem cor dispara `fetchNodeConnections` na hora (sem "Ver conexões"). Grifo sem cor continua abrindo só a barra.
- [ ] Sobre a barra flutuante aparece "conexão i de N" com ◀ ▶ (e ← → no teclado com a barra focada). Cada passo aplica `irAte` no destino e o destaca (grifo secundário forte, na cor de origem). Sem conexão: "nenhuma conexão acima do limiar".
- [ ] A sidebar mostra, para a conexão atual, título da seção, "p. N", os termos casados e o primeiro parágrafo da seção destino (texto), com botão "Ir até" redundante.
- [ ] As linhas SVG continuam (12 mais fortes), e a linha da conexão atual fica realçada.

Estado de visualização no servidor
- [ ] Tabela `workspace_view_states` (`user_id, collection_id, state jsonb, updated_at`, única em `(user_id, collection_id)`). `state` = `{viewport: {x, y, escala}, quadros: {docId: {x, y}}, apresentacao: {docId: {modo, paginaAtual}}, marcadoresVisiveis: bool, documentoFocado?: id}`.
- [ ] `GET/PUT /documents/collections/:id/view-state`, escopo por usuário via coleção. Repositório com upsert; usecase valida tamanho (≤ 64 KB) e formato mínimo.
- [ ] Frontend: carrega o estado junto com o contexto; aplica antes do primeiro paint do canvas (sem "pulo"); salva com debounce de 800 ms depois de qualquer mudança, e ao sair da página (`visibilitychange`/`pagehide`). `localStorage` deixa de ser usado para posições, modo e página (migração: se houver estado local e nenhum no servidor, envia o local uma vez).
- [ ] `?doc=` na URL continua valendo e tem prioridade sobre o estado salvo para o foco.

Marcadores
- [ ] Botão na barra de zoom do canvas: "Marcadores de seção" (ícone), alterna e persiste no estado de visualização.
- [ ] Indexador: linha dominada por números não vira seção (`595ebc4`); o calendário reindexado deve cair de 59 para menos de 15 seções.

Testes
- [ ] Backend: integração do repositório (upsert, isolamento entre usuários, coleção alheia 404); unitário do usecase (estado grande → 400).
- [ ] Frontend unit: navegador (i de N, limites, teclado, chama irAte), estado aplicado sem efeito visível, debounce de salvamento, marcadores escondidos.
- [ ] E2E mock: clique em grifo colorido dispara conexões e ◀ ▶ percorrem; GET view-state com página 4 → quadro abre na página 4 e viewport restaurado; botão esconde marcadores e PUT sai com `marcadoresVisiveis: false`.
- [ ] E2E real: folhear até a p. 4, novo contexto com o mesmo token, reabre em p. 4.
- [ ] Reversão em cada asserção nova.

## Non-Goals

- Sincronizar estado entre abas em tempo real. Histórico de estados. Estado por dispositivo.
- Conexões entre parágrafos (a unidade continua a seção do nó de sumário).
- Editar o sumário na interface.

## Constraints

- Padrões iguais aos das marcas (WA1/WA2): modelo + `ManagedModels()`, repositório gorm, usecase, handler com `parseUintParam`, rotas no grupo `documents`, BFF em `documents/collections/[id]/view-state/route.ts`, cliente em `collections.ts` com `redirectOnUnauthorized: false`.
- O estado é preferência, não dado da tese; nada dele entra no caminho de recuperação.
- Aplicar o estado antes do primeiro paint exige que o `WorkspaceCanvas` receba o estado como prop (carregado em `page.tsx` junto com o contexto), não o carregue por efeito.
- Salvar com debounce e `keepalive: true` no fetch de saída.
- `PdfFrame` já aceita `apresentacao`; `useViewport` precisa aceitar viewport inicial e expor uma forma de setar.

## Approach

1. Backend: `WorkspaceViewState` + rotas + testes (mesmo desenho das marcas).
2. BFF + cliente + contrato tolerante.
3. `page.tsx` carrega contexto, marcas, legenda e estado em paralelo e passa tudo ao canvas; `WorkspaceCanvas` inicializa `viewport`, `posicoes`, `apresentacoes`, `marcadoresVisiveis` das props; `useViewport({inicial})`.
4. `useSalvarEstado` com debounce e `pagehide`.
5. `ConnectionNavigator.tsx` sobre a barra; clique em grifo colorido dispara conexões; sidebar com detalhe da conexão atual.
6. Botão de marcadores; E2E e reversões.

## Open Questions

- Migração do `localStorage`: uma vez, silenciosa. — *decidido por padrão*.
- Se o estado salvo referenciar documento que saiu da coleção, ignora a entrada. — *decidido por padrão*.

## Decisões tomadas na entrevista (2026-09-08)

- Estado de visualização no backend, junto das marcas.
- Clique em grifo colorido busca conexões e abre o navegador i de N.
- Marcadores: corrigir o indexador e dar botão para esconder.
- Corpus ampliado com matriz e ementas de SI (Palmas); documentos digitalizados descartados.

## Ambiguity Report (grill-me, modo spec, 2026-09-08)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (limiar de 15 secoes no calendario e chute inicial)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (estado no servidor x localStorage decidido; tabela x metadata da colecao nao comparados -- tabela pela unicidade por usuario)
  Assumptions:  0.25  ✓ clear (aplicar estado antes do primeiro paint depende de mover o carregamento para page.tsx)
  ──────────────────────────────
  Aggregate:    0.15  ✓ below threshold (0.2 spec)
```

## Discoveries

(vazio)
