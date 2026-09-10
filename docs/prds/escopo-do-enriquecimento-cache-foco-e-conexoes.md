# Escopo do enriquecimento por toggles, cache por hash, foco na fonte e conexões legíveis

**Status:** entregue (2026-09-10), branch `feat/conexoes-por-token` (PR #72, sem merge)
**Origem:** o autor: "separar a indexação em toggles (tudo resumido, cada pdf, cada página, cada tópico); clicar na fonte e ir ao lugar específico com o texto grifado em cor diferente, com foco na tela; conexões mais semânticas, mais fáceis de enxergar e de se locomover". E o item 2 da lista anterior: reindexar não deveria pagar o enriquecimento de novo.

## Goals

1. **Escopo do enriquecimento por usuário.** Três toggles em Configurações: *Sinopse, tópicos e árvore do documento*; *Resumo e tópicos por página*; *Resumo por seção*. Padrão: os três ligados. O indexador lê as preferências do dono do documento e só gasta LLM no que está ligado; o que está desligado cai no fallback determinístico e fica registrado em `enrichment_scope`.
2. **Cache do enriquecimento por hash.** Resumo e tópicos de página, e resumo de seção, ficam guardados por `(tipo, modelo, sha256 do texto)`. Reindexar um documento cujo texto não mudou não chama a LLM para páginas nem seções; só para sinopse, sinais e árvore. Metadados `enrichment_cache_hits_pages` e `enrichment_cache_hits_sections`.
3. **Foco na fonte.** Clicar numa fonte do chat (ou ir a uma conexão) centraliza, aproxima o zoom até o trecho ficar legível e acende um holofote (escurece o resto por alguns segundos). O grifo da citação continua em rosa, distinto da seleção (âmbar) e das marcas do usuário.
4. **Conexões legíveis.** Tipo novo `mesma-disciplina` (os dois trechos citam o mesmo código de disciplina, ex. SIS07302); cor por tipo (norma, disciplina, termo raro) na linha, no rótulo e na lista; ao ir para uma conexão, as linhas do destino que contêm os termos em comum ganham o grifo rosa; setas ← → percorrem as conexões e Esc limpa.

Pronto quando: (1) toggle desligado zera as chamadas daquele escopo numa reindexação (medido nos metadados); (2) segunda reindexação do PPC de SI com o mesmo texto faz ≤ 4 chamadas de LLM (era 48); (3) Vitest do canvas prova zoom e holofote no clique da fonte; (4) unitário de `BuildNodeConnections` prova `mesma-disciplina`, e o E2E com mock cobre a cor por tipo e o teclado.

## Requirements

Escopo
- [x] `models.UserIndexingPreferences` (user_id único; três booleanos), `domain.IndexingPreferences` + `DefaultIndexingPreferences()`, repositório GORM, `GET/PUT /user/indexing-preferences`.
- [x] `LLMEnricher.EnrichWith(ctx, llm, doc, result, prefs)`; `Enrich` chama com o padrão. Documento desligado: árvore determinística, sinopse fallback (`synopsis_source=disabled`), sem sinais. Páginas/seções desligadas: sem lotes.
- [x] Configurações: card "O que a IA gera na indexação" com os três switches; salva no PUT; i18n nos três idiomas.

Cache
- [x] `models.EnrichmentCacheEntry` (kind, model, content_hash únicos; summary, topics). `domain.EnrichmentCache` (Lookup/Store). Hash = sha256(versão do prompt + texto). Página: texto da página. Seção: título + texto próprio da seção (o mesmo do lote).
- [x] O enricher consulta antes de montar os lotes e grava depois de cada lote bem-sucedido. Sem cache configurado, nada muda.

Foco
- [x] `irAte(..., { foco: true })`: escala tal que o rect ocupe ~55% da largura da tela (entre 0,9 e 1,8), centralização, e `holofote` (rect + expiração) desenhado no mundo com `box-shadow` gigante, some em 2,5 s ou ao primeiro pointerdown.

Conexões
- [x] Backend: `disciplineCodes` (regex `\b[A-Z]{2,4}\d{4,6}\b`), aresta `mesma-disciplina` valendo 2 por código, como `mesma-norma`. DTO/contrato aceitam o tipo.
- [x] Frontend: `corPorTipo`, legenda na lista, grifo das linhas do destino com os termos em comum ao navegar, teclado ← → Esc quando há conexões carregadas.

## Non-Goals

- Toggles por documento ou por coleção (só por usuário).
- Invalidar cache por modelo diferente (é chave; modelo novo = cache novo).
- Conexões por embedding ou por LLM.

## Constraints

- O indexador e a API compartilham o módulo Go; migração é `AutoMigrate` em `db.ManagedModels()`.
- `observedLLMService` envolve o cliente; o nome do modelo vem de `domain.LLMMetadataProvider`.
- Booleano com `default:true` no GORM grava `true` quando o valor é `false`; os campos ficam sem default e o upsert escreve as três colunas.

## Decisões sem entrevista

- Cache global (não por usuário): o texto de uma página é o mesmo para qualquer usuário; o modelo faz parte da chave.
- Preferências por usuário, não por coleção: é o que o autor pediu ("toggles") e o que o provedor já faz (`enabled_for_indexing`).
- Holofote por CSS, sem biblioteca; 2,5 s é chute registrado, ajustar com uso.

## Discoveries

- **Cache medido na stack real.** Lei 11.788 (13 p.): 5 → 2 chamadas de enriquecimento na segunda reindexação. PPC de SI (210 p., 144 seções): 51 → 4 requisições à LLM, 198 mil → 73 mil tokens de entrada; 208/208 páginas e 144/144 seções vieram do cache. O que sobra (73 mil) é a árvore, a sinopse e os sinais, que leem o documento inteiro e não têm cache.
- **Toggles verificados na API e na tela:** `PUT /user/indexing-preferences` grava e `GET` devolve; o card em Configurações (aba de IA) alterna e salva na hora. O indexador lê as preferências do dono do documento a cada indexação e grava `enrichment_scope`.
- **Foco.** A primeira versão tinha um timer de 2,5 s e não aparecia no E2E porque o documento ainda não tinha sido medido quando a fonte foi clicada: o pedido de foco agora fica pendente até as medidas chegarem e é refeito; e o holofote fica até o primeiro gesto (clique ou roda), sem timer. Em "uma página", o rect do parágrafo é calculado "como mostrada" para a página destino, então o foco vale mesmo trocando de página.
- **Conexões.** `mesma-disciplina` por código de componente (regex `\b[A-Z]{2,4}\d{4,6}\b`), com a mesma força de `mesma-norma`; cor por tipo na linha, no rótulo e na lista; ao ir a uma conexão, as linhas do destino que contêm os termos em comum ganham o grifo rosa; ← → percorrem, Esc limpa.
- **Contador `enrichment_llm_calls`** era gravado antes da chamada de sinais e ficava um a menos; corrigido.
- **Esquema:** duas tabelas novas (`user_indexing_preferences`, `enrichment_cache_entries`); DBML regenerado, 25 tabelas.
