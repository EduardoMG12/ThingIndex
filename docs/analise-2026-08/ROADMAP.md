# Roadmap ThingIndex — consolidação da rodada de análise (2026-08-17)

> Produzido a partir de quatro auditorias paralelas (backend Go, frontend, gap
> artigo × código, benchmark científico) mais verificação direta no repositório.
> Relatórios completos em `docs/analise-2026-08/`.
>
> **Regra deste documento:** só entra afirmação que foi verificada rodando
> comando ou lendo o código. Onde uma auditoria errou, o erro está registrado.

---

## 0. Ponto de partida (verificado, não presumido)

```
go build ./...            → limpo
go vet ./...              → limpo
go vet -tags=integration  → limpo
gofmt -l .                → vazio
go test ./...             → tudo passa
npm run lint              → 0 erros, 3 warnings
npm run typecheck         → limpo
npm run test:unit         → 50 arquivos, 134 testes, todos passam
npm run build             → 61 páginas geradas
docker compose config     → válido
```

Backend 45k linhas Go, 53 arquivos de teste. Frontend 100 componentes, 50
arquivos de teste, 14/14 E2E. CI com três jobs. Zero TODO/FIXME.

**A higiene está resolvida.** O que resta não aparece em ferramenta — aparece em
carga, em concorrência, em acessibilidade e na arguição da banca.

---

## 1. Bugs que eu verifiquei linha a linha

### 1.1 P0 — o outbox não reivindica nada, e o documento pode ser indexado 2×

`api/internal/repository/gorm_outbox_repository.go:22-53`

A função se chama `ClaimPendingEvents` e usa `FOR UPDATE SKIP LOCKED`. Mas o
`UPDATE` final só faz isto:

```go
Updates(map[string]any{
    "attempts":   gorm.Expr("attempts + 1"),
    "updated_at": now,
})
```

O `status` **continua `PENDING`** e o `available_at` **não avança**. O lock morre
no commit da transação — e a publicação no RabbitMQ só acontece **depois**, em
`outbox_dispatcher.go:46-70`. Entre o commit e o publish há uma janela em que o
evento está visível e reivindicável por qualquer outro dispatcher.

Duas consequências reais:

- Com **duas réplicas** do indexer (nada no compose impede escalar), o mesmo
  evento é publicado duas vezes → `Process` roda 2× no mesmo documento → custo
  de LLM dobrado e `ReplaceIndexingResult` concorrente.
- Com **uma réplica**: se `MarkEventPublished` falhar, o `return` aborta o batch
  e o evento já publicado segue `PENDING` — republicado a cada 5s, indefinidamente.

O teste de integração existente só verifica filtro e incremento de `attempts`,
por isso o defeito passa há muito tempo.

Correção — marcar dentro da mesma transação do claim:

```go
Updates(map[string]any{
    "status":       domain.OutboxEventStatusInFlight,
    "attempts":     gorm.Expr("attempts + 1"),
    "available_at": now.Add(visibilityTimeout),   // reentrega se o worker morrer
    "updated_at":   now,
})
```

E tornar o consumidor idempotente: descartar evento cujo documento já esteja
`INDEXED` com `indexed_at` posterior ao evento.

> Este é o item nº 1 da lista inteira. É o único que **corrompe dado e queima
> dinheiro** em silêncio, e é o mais fácil de a banca provocar ao vivo pedindo
> para escalar o worker.

### 1.2 P0 — validação de lista é código morto (falta `dive`)

`api/internal/dto/catalog.go:22-47`

```go
type StoreParagraphNodesRequest struct {
    Nodes []ParagraphNodeInput `json:"nodes" binding:"required"`   // sem dive
}
```

Sem `dive`, o validator **não desce nos elementos do slice**. As constraints de
`ParagraphNodeInput` (`gte=0`, `gte=1`, `required`) nunca são avaliadas. Vale
igual para `StoreTableSectionsRequest` e `StoreMetadataPairsRequest`.

Resultado: dá para gravar `sequence_order` negativo e `content` vazio direto nas
tabelas estruturadas — exatamente os campos de que a ordenação
(`ORDER BY sequence_order`) e a proveniência dependem. Para um trabalho cuja
tese é auditabilidade, é o pior lugar possível para aceitar lixo.

Correção: `binding:"required,dive"`. Atenção ao efeito colateral: `required` num
`int` rejeita `0`, que é o valor legítimo do primeiro item — trocar por `gte=0`
sozinho.

### 1.3 P0 (frontend) — três defeitos que a banca vê na primeira tela

Verificados no código e no build de produção:

| Defeito | Arquivo | Efeito |
|---|---|---|
| `<html lang="en">` fixo | `src/app/layout.tsx:56` | Página em espanhol com `lang="en"`. Falha WCAG 3.1.1 (Nível A). Leitor de tela pronuncia "Contraseña" com fonética inglesa. Google indexa 61 páginas como inglês |
| Splash de 3,5 s com o `setItem` comentado | `src/components/ui/SplashScreen.tsx:99` | O intervalo de 3 dias está morto: **toda** visita paga 3,5 s de overlay `z-[9999]` sem `pointer-events-none`. O E2E injeta a chave no localStorage para contornar — o teste documenta o bug em vez de pegá-lo |
| Tema escolhido não é persistido | `src/components/ui/ThemeToggle.tsx:37-41` | Zero escrita em localStorage/cookie. O toggle se desfaz em todo F5 |

E um quarto, que interage com o terceiro: no **tema claro**, `--primary` sobre o
background dá **2.16:1** e `--primary-foreground` branco sobre `bg-primary` dá
**2.32:1** — ambos reprovam AA (exige 4.5:1). No tema escuro o mesmo par dá
9.89:1. Ou seja: o usuário que cai no tema claro vê todo botão primário quase
ilegível, tenta trocar para escuro, e o F5 o joga de volta.

Uma demo ao vivo em máquina com `prefers-color-scheme: light` mostra isso na
primeira tela.

### 1.4 P1 — os que valem corrigir, sem urgência de P0

| # | Item | Onde | Efeito |
|---|---|---|---|
| 1 | Consumidor sequencial, sem timeout por mensagem | `queue/rabbitmq.go:204-256` | O `ctx` do handler é o do processo, sem deadline. Um PDF grande ou um LLM lento trava a fila inteira. Não há reaper para documentos presos em `PROCESSING` |
| 2 | SSE cortado pelo `WriteTimeout` de 30 s | `cmd/api/main.go:58-64` | `WriteTimeout` é deadline absoluto, não idle. Resposta de chat >30 s é truncada — e o texto parcial **não é persistido** |
| 3 | Rate limit só em `/auth`, chaveado por IP | `middleware/rate_limit.go:75` | O frontend chama a API server-side, então o IP é sempre o do container: um balde global para todos os usuários. O 11º login do minuto leva 429. Upload e chat não têm limite nenhum |
| 4 | Contexto da coleção inteiro em RAM | `gorm_collection_context_repository.go:48-90` | 6 `Find` sem `LIMIT`, com o texto integral. Chamado em toda mensagem de chat. Uma coleção grande derruba o processo por OOM |
| 5 | Erros de estado engolidos no worker | `processing/document_processor.go:47-56` | `_ =` em ~10 transições, inclusive as que marcam `FAILED`. Documento congela em `PROCESSING` sem sintoma |
| 6 | Corte de string em bytes | `chat_usecase.go:277`, `gorm_chat_repository.go:155` | `normalized[:255]` quebra UTF-8 no meio do code point. Em português é o caso comum. O resto do código já faz certo com `[]rune` |
| 7 | Soft delete deixa conteúdo no banco | `gorm_catalog_document_repository.go:469-489` | Falta `Unscoped()`. O texto integral do documento "excluído" permanece na tabela. Os caminhos de reindexação usam `Unscoped()` — incoerência interna |

### 1.5 Onde procuramos e **não** achamos problema

Registro explícito, para não inflar a lista e para você poder responder à banca
com segurança:

- **Multi-tenancy / IDOR:** revisadas as funções de repositório alcançáveis por
  rota. Todas filtram por `user_id`. `gorm_structured_data_repository.go` não
  tem escopo próprio (13 `Where`, zero `user_id`), mas os dois únicos chamadores
  (`GetDocumentIndex`, `GetDocumentMonitoringSummary`) validam posse antes.
  **Não há vazamento** — mas a proteção é convenção, não estrutura. Um teste que
  trave isso vale mais que um comentário.
- **SQL injection:** GORM parametrizado em toda parte. `escapeLikePattern` trata
  `%`/`_`.
- **Path traversal:** o nome do objeto é gerado pelo servidor (timestamp + UUID +
  extensão de allowlist). O nome do cliente nunca chega ao storage.
- **SSRF:** `netsec.StorageEndpointValidator` resolve DNS, bloqueia faixas
  privadas, e o `DialContext` customizado impede rebinding.
- **Criptografia:** AES-GCM, chave derivada por usuário via HMAC-SHA256, nonce
  novo por operação, com teste de isolamento entre usuários. Correto.
- **AutoMigrate:** as três listas (produção, teste de integração, E2E) conferem —
  21 modelos em todas. A divergência relatada no PR #4 está resolvida.

---

## 2. O gap que decide a nota

Aqui o assunto muda. Os bugs acima são engenharia; o que vem agora é o trabalho.

### 2.1 O diagnóstico em uma frase

**Existe um artefato de engenharia forte e uma pesquisa fraca.** Não por falta de
trabalho — por falta de comparação.

O artigo afirma, em quatro lugares diferentes, superioridade em auditabilidade,
determinismo e custo sobre RAG vetorial. E **nunca compara com nada**. O único
"comparativo" no código (`flat-page-control`, `flat-hybrid-control`) compara o
sistema com variações do próprio sistema. O código admite a lacuna
explicitamente em `indexing_benchmark.go:808`:

> *"This repository does not ship a native vector index in the current runtime"*

### 2.2 A pergunta de pesquisa não é falseável

> *"É possível sustentar um sistema de inteligência documental com
> auditabilidade utilizando exclusivamente modelagem relacional?"*

Resposta: sim, o sistema está de pé. Fim. Nenhuma banca aceita como hipótese algo
que a existência do artefato já responde.

Reformulação sugerida — não-inferioridade, que é testável:

> **H1 (precisão):** em corpora documentais estruturados de até 1.000 documentos,
> a recuperação relacional é **não-inferior** à densa com pgvector em nDCG@10,
> com margem δ = 0,05.
>
> **H2 (auditabilidade):** a relacional atinge taxa de proveniência completa
> ≥ 99% e determinismo de 100% em repetições; a densa fica abaixo em ambas.
>
> **H3 (custo):** existe um tamanho de coleção M abaixo do qual o custo por
> consulta da relacional é menor que o da densa, e acima do qual é maior.

H3 escrita assim é o movimento mais inteligente disponível: transforma uma
fraqueza em **achado**. "Existe um ponto de inversão M" é um resultado; "somos
mais baratos" é propaganda.

### 2.3 O benchmark não discrimina — e eu provei isso

Reportar `structure=100 / node_top1=100 / exact=100` **é evidência de que a
métrica não discrimina**, não de que o sistema é bom. Verifiquei:

| Item | Valor real |
|---|---|
| Documentos no corpus | **5** (3 texto/HTML inline, 2 PDFs de ~2 KB) |
| Consultas totais | **11** (6 de documento + 5 de coleção) |
| Documentos escaneados | **0** — o caminho de OCR nunca é medido |
| Tabelas em PDF | **0** — só há 1 tabela, em HTML |
| Páginas máximo | **2** |
| Casos negativos | **0** — nunca se mede falso positivo |
| Distratores | ~0 |

E dois achados que eu confirmei rodando script sobre o `corpus.json`:

**Circularidade do gabarito: 6 de 6.** Todo `expected_top_node_title` é cópia
literal de um item de `expected_outline_titles` do mesmo documento. O gabarito de
recuperação é definido em função da saída estrutural do próprio pipeline. Se o
extrator renomear `"3 DISPOSITIVO"` para `"3. Dispositivo"`, a métrica quebra —
não porque a recuperação piorou, mas porque a **rotulação** mudou.

**O scorer é contagem de substring** (`document_usecase.go:2136`):

```go
for _, term := range queryTerms {
    if strings.Contains(normalized, term) { score++ }
}
```

Sem IDF, sem normalização por comprimento, sem BM25. E as consultas foram
escritas copiando palavras do trecho-alvo: `"payment deadline invoice receipt"`
para um texto que diz *"Invoices must be paid within 15 days of receipt"*.

O teste é, portanto: *"um matcher de substring encontra o parágrafo cujas
palavras eu copiei"*. Um `grep` acerta 100%. É tautológico.

Some-se o que o próprio `estado-do-projeto.md` já registra com honestidade: parte
do salto `60/66,7/81,8 → 100/100/100` veio de **corrigir uma fixture**, não de
melhorar o pipeline.

> **Isso não é vergonha — é material de tese, se apresentado como tal.** A
> leitura correta: *"a sensibilidade da métrica a um artefato de renderização do
> corpus demonstra que ela é dominada pela camada de extração de texto, não pela
> lógica de indexação"*. Isso justifica cientificamente separar erro de extração,
> de estruturação e de recuperação, e medir cada camada isoladamente. E o commit
> isolado (`3dac860`) permite reportar a tabela antes/depois como **evidência de
> sensibilidade da métrica**.

### 2.4 Correção de uma auditoria: o custo do chat é O(1) em tokens, não O(n)

Uma das auditorias afirmou que o chat envia o contexto inteiro da coleção ao LLM,
tornando o custo O(n) e derrubando a tese de "custo previsível". **Fui verificar
e a afirmação está errada.**

`buildCollectionChatPrompt` (`chat_usecase.go:364-446`) trunca tudo:

- fontes: no máximo 6, com snippet cortado
- inventário de documentos: `if index == 6 { break }`, resumo cortado em 220 chars
- hits de outline: `if index == 5 { break }`, resumo cortado em 180 chars
- histórico: `limitChatHistory(..., 6)`, cada mensagem cortada em 500 chars

O prompt é **limitado por construção**. O custo de LLM por consulta é O(1) no
tamanho da coleção. A tese de custo previsível **se sustenta** nesse eixo.

O custo O(n) real está em outro lugar, e é o item 1.4-4: `GetCollectionContext`
carrega todas as páginas, parágrafos, tabelas, chunks e metadados da coleção em
RAM — e o chat chama isso a cada mensagem, para depois usar 6 fontes. É
desperdício de memória e I/O, não de tokens.

**Consequência prática:** a correção certa é empurrar a seleção para o banco
(`tsvector` + `LIMIT`, carregando conteúdo só do top-N). Isso mata o risco de OOM
e ainda dá um número bonito para o artigo: *"a recuperação relacional consulta a
coleção inteira em 9 consultas de custo constante e materializa apenas o top-N"*.

> Registro isto porque é exatamente o tipo de erro que, se entrasse no artigo sem
> conferência, a banca desmontaria em trinta segundos abrindo o código.

### 2.5 Afirmações do artigo × realidade do repositório

| Afirmação | Estado |
|---|---|
| "20 tabelas" | São **21** no `AutoMigrate` (produção, integração e E2E conferem entre si) |
| "14 classes" no diagrama | O `classes.mmd` tem **16**; o código tem 21 |
| RF15: i18n PT-BR e EN-US | Frontend suporta **3** locales (`pt-BR`, `en`, `es`); backend só resolve **pt** e **en** (`i18n/messages.go:192-197`) — `Accept-Language: es` cai em inglês enquanto a UI mostra espanhol |
| Apêndice A: "esquema em DBML no repositório" | **Nenhum arquivo `.dbml` existe.** `find . -name "*.dbml"` → vazio |
| "o repositório será mantido público" | É privado (registrado no `estado-do-projeto.md`) |
| Cronograma "junho a dezembro de 2026" | O sistema já está implementado, com CI verde |
| RNF03: "centenas de documentos sem degradação" | **Nunca medido.** Só existe contagem de consultas com 1 vs 20 documentos |
| "cobertura de testes automatizados" | Não há `-coverprofile` no CI — não existe percentual publicável |

E um achado que compromete o próprio backup do artigo:

**O `references.bib` local está dessincronizado do Overleaf.** A chave
`schymik2015benefits` é citada na **primeira frase da introdução** e não existe
no `.bib` — mas o PDF compilado renderiza `[Schymik et al. 2015]` corretamente e
lista a referência completa. Ou seja: compilar o `.md` versionado hoje produz um
artigo com citação quebrada. O PR #9 versionou os arquivos para servir de backup;
o backup está incompleto.

Correção: puxar o `references.bib` atual do Overleaf e commitar. Verificação:

```bash
grep -oP '\\cite\{\K[^}]+' docs/article/article-overleaf.md | tr ',' '\n' | tr -d ' ' | sort -u > /tmp/cited.txt
grep -oP '^@\w+\{\K[^,]+' docs/article/references.bib | sort -u > /tmp/bib.txt
comm -23 /tmp/cited.txt /tmp/bib.txt   # deve sair vazio
```

Vale virar step do CI: uma citação órfã quebra a compilação na véspera da entrega.

### 2.6 Trabalhos relacionados são produtos, não pesquisa

A seção cobre Notion, LangChain e LlamaIndex. Nenhum é trabalho acadêmico;
nenhum tem paper revisado por pares. É o defeito mais visível do artigo — um
avaliador rigoroso desconta antes de olhar o resto.

Mínimo a incorporar:

| Área | Trabalho | Por quê |
|---|---|---|
| RAG estruturado | **GraphRAG** — Edge et al. (2024) | concorrente direto da tese; o artigo diz "grafos" e nunca cita o trabalho canônico |
| Retrieval denso | **DPR** — Karpukhin et al. (2020) | fundamento do que o artigo chama "RAG convencional" |
| Late interaction | **ColBERT / v2** — Khattab & Zaharia (2020) | a banca **vai** perguntar "por que não ColBERT?" |
| Esparso neural | **SPLADE** — Formal et al. (2021) | mostra que léxico ≠ ultrapassado — **argumento a favor da sua tese**, hoje desperdiçado |
| Fusão | **RRF** — Cormack et al. (2009) | base do braço híbrido |
| Avaliação | **BEIR** — Thakur et al. (2021) | comparabilidade internacional |
| Avaliação de RAG | **RAGAS** — Es et al. (2023) | faithfulness/groundedness é exatamente o que "auditabilidade" quer capturar |
| Proveniência | **Cheney, Chiticariu & Tan (2009)** | Buneman sozinho é raso para uma tese que **se chama** auditável; o tripé why/how/where é o vocabulário que falta |
| Proveniência | **W3C PROV-DM** (2013) | padrão de modelagem |
| Estrutura de PDF | **Nougat** (2023) / **PageIndex** | o código **já compara** com exports do `pageindex` e o artigo não menciona — evidência jogada fora |

Também falta: seção de **Ameaças à Validade** (obrigatória em trabalho empírico)
e a delimitação honesta de onde a abordagem perde. Essa última já existe escrita
— **comentada na linha 844** do `.md`:

> *"em domínios abertos e corpus massivos, embeddings provavelmente superarão
> joins hierárquicos em flexibilidade de matching"*

Traga para o corpo. Trabalho que declara seu limite é lido como maduro.

---

## 3. O experimento que falta

Núcleo do TCC II. Quatro braços, mesmas consultas (pareado):

| Braço | O quê | Implementação |
|---|---|---|
| **A** | ThingIndex relacional | já existe |
| **B** | BM25 | `tsvector` + `ts_rank_cd` com dicionário `portuguese`. **Obrigatório**: o artigo cita Robertson & Zaragoza e depois não usa BM25 |
| **C** | Denso | `pgvector/pgvector:pg16` (drop-in no compose), `multilingual-e5-base` (768d, roda local, sem custo de API) |
| **D** | Híbrido RRF | estado da prática; sem ele a comparação está datada |

Mais um **fator ortogonal de chunking** (unidades estruturais do ThingIndex vs.
512 tokens cego). Rodar C e D nos dois regimes **isola a contribuição da
estrutura** — permite afirmar "a estrutura ajuda" com evidência em vez de
retórica. Isso sozinho já é contribuição publicável.

**Corpus** (público, obtenível legalmente no Brasil), ~60 documentos em 5
estratos: acórdãos STF/STJ, DOU, editais de licitação (para tabelas),
**escaneados** (para OCR — hoje sem medida nenhuma) e SciELO/arXiv.

Truque de baixo custo: gerar os escaneados **imprimindo e digitalizando** os
documentos nativos. O texto original vira gabarito de OCR perfeito, custo zero,
e permite variar dpi (150/300/600) como variável controlada → curva CER × dpi,
gráfico excelente e barato.

**Gabarito sem circularidade** — as três regras que salvam o trabalho na arguição:

1. O gabarito estrutural sai do **PDF**, nunca da saída do sistema. Comparação
   por Levenshtein normalizada ≥ 0,90 com normalização declarada, não por
   igualdade de string.
2. As consultas nascem da **necessidade de informação**, não do texto-alvo. O
   anotador lê, **fecha o documento**, e só então escreve. Três variantes:
   literal, paráfrase, ruidosa. **A queda de P@1 de literal → paráfrase → ruidosa
   é o gráfico mais forte do artigo** — mede exatamente o que o benchmark atual
   esconde.
3. Dupla anotação em ≥20% do corpus, com **Cohen's κ** reportado (piso ~0,70).
   Sem isso, "gabarito" é opinião do autor.

Mais: distratores obrigatórios e ~10% de casos negativos (resposta correta =
"nenhum documento relevante"), para medir falso positivo.

**Métricas** — trocar o `exact_match` agregado (que hoje soma unidades
incomparáveis: acerto de nó com acerto de documento) por painel padrão: P@1/@3/@5,
R@5/@10, **MRR** e **nDCG@10** com relevância graduada (0-3). Para estrutura:
F1 de seções, erro de fronteira de página, **TEDS** para tabelas (contínuo em
[0,1] — dá média, desvio, boxplot e teste; `expected_table_count == n` dá um
booleano). Para OCR: CER e WER.

**Auditabilidade** — três métricas novas, que são contribuição própria:

- **TPC** (taxa de proveniência completa): % de itens com `document_id`,
  `page_number` e `sequence_order` todos não nulos e resolvíveis. Mede
  diretamente o furo: `ParagraphNode.PageNumber` é `*int`, **nulável**, e blocos
  do tesseract entram sem `Box`.
- **TCV** (taxa de citação verificável): % de fontes citadas cujo span aparece
  literalmente no parágrafo apontado.
- **Determinismo**: mesma consulta 30×, hash idêntico. **É a vitória mais barata
  disponível** — o relacional dá 100%, o denso com HNSW varia com `ef_search` e
  ordem de inserção. Está sendo deixada na mesa.

**Estatística**: teste pareado por consulta (Wilcoxon signed-rank, não assuma
normalidade), Bonferroni para as comparações múltiplas, IC 95% por bootstrap,
Cliff's delta para tamanho de efeito. Determinismo do pipeline é vantagem, não
problema — declare que a incerteza reportada é de **amostragem sobre as
consultas**, estimada por bootstrap, e não rode 30× um pipeline determinístico
fingindo que gera variância.

---

## 4. Ordem sugerida

Ordenado por impacto na nota dividido por esforço.

### Fase 0 — 2 dias, destrava tudo

| # | Item | Por quê | Estado |
|---|---|---|---|
| 1 | `references.bib` dessincronizado (§2.5) | O backup do artigo está quebrado **agora**. Risco de perda real | ✅ feito |
| 2 | `dive` nos 3 DTOs (§1.2) | Uma linha por struct. Fecha entrada de lixo nas tabelas da tese | ✅ feito |
| 3 | Splash + `html lang` + persistir tema (§1.3) | Três correções pequenas que a banca vê na primeira tela | ✅ feito |
| 4 | Contraste do tema claro (§1.3) | Ajuste de dois tokens em `globals.css` | ✅ feito |
| 5 | Conferir os números do artigo (21 tabelas, idiomas, DBML) | Detalhe calibra a confiança da banca no resto | ⬜ pendente |

#### O que foi aplicado nesta rodada

- **`api/internal/dto/catalog.go`** — `binding:"required,dive"` nos três DTOs de
  escrita em lote, e `required,gte=0` → `gte=0` no `SequenceOrder` (a tag antiga
  rejeitava `0`, que é o valor legítimo do primeiro nó).
- **`api/internal/dto/catalog_binding_test.go`** — 9 casos de regressão.
  Verificado RED → GREEN: revertendo o `dive`, o teste falha apontando
  exatamente os payloads inválidos que passavam.
- **`frontend/src/components/ui/SplashScreen.tsx`** — `setItem` descomentado. O
  intervalo de 3 dias volta a valer; some o overlay de 3,5 s em toda visita.
- **`frontend/src/components/ui/ThemeToggle.tsx`** + **`src/app/layout.tsx`** —
  tema persistido em `localStorage`, com o script de boot lendo a preferência
  salva antes de cair no `prefers-color-scheme`.
- **`frontend/src/app/layout.tsx`** — `document.documentElement.lang` corrigido
  no script pré-paint a partir do segmento de locale do pathname.
- **`scripts/check-article-citations.sh`** — guard que quebra quando há citação
  sem entrada no `.bib`. Pegou o `schymik2015benefits` na primeira execução.
- **`frontend/src/app/globals.css`** — `--primary` de 48% → 31% e `--accent` de
  52% → 34% no tema claro. Contrastes recalculados pela fórmula WCAG 2.x
  (luminância relativa): os quatro pares saem de 2.16/2.32/2.39/2.56:1 para
  4.77/5.13/4.93/5.29:1, todos acima do piso AA de 4.5:1. O tema escuro não foi
  tocado — já dava 9.89:1.

> **Trade-off que descobri corrigindo o `lang` — vale registrar.** A correção
> "certa" (root layout `async` chamando `getServerLocale()`) funciona, mas
> `cookies()` opta a rota inteira para renderização dinâmica: medi a queda de
> **38 páginas pré-renderizadas para 1**. Preço alto demais para um atributo.
>
> A raiz do problema é estrutural: o `<html>` mora em `app/layout.tsx`, que é
> comum a todos os locales, enquanto o locale só existe em `app/[locale]/`. A
> correção definitiva é mover o `<html>`/`<body>` para `app/[locale]/layout.tsx`
> — que já recebe `params.locale` e já tem `generateStaticParams`, então
> entregaria `lang` correto **no HTML servido**, com SSG intacto. É uma
> refatoração de rotas, não um one-liner, e por isso ficou para a Fase 1.
>
> O que está aplicado agora é o meio-termo honesto: o HTML servido ainda sai com
> `lang="pt-BR"`, mas o script pré-paint corrige o DOM antes da primeira
> pintura. **Leitor de tela e corretor ortográfico leem o DOM vivo, então o
> problema de acessibilidade (WCAG 3.1.1) está resolvido na prática.** O que
> permanece é o crawler que não executa JS — ou seja, o efeito de SEO. Registro
> a limitação em vez de fingir que a correção é completa.


### Fase 1 — 1 semana, tira o risco de corrupção

> **Armadilha de ambiente que reencontrei rodando o E2E — leia antes de rodar.**
> A suíte falhou em 4 testes na primeira tentativa. **Não era regressão minha:**
> a porta 3000 estava ocupada por outro projeto (`urso-azul/totinho`), e com
> `reuseExistingServer: !CI` o Playwright reaproveitou aquele servidor e testou
> a aplicação errada. É exatamente o problema já documentado na seção 2.5 do
> `estado-do-projeto.md` — o comentário no `playwright.config.ts` inclusive
> avisa sobre isso.
>
> A config já prevê a saída. **Sempre rode local com porta explícita:**
> ```bash
> cd frontend && THINGINDEX_E2E_PORT=3457 npx playwright test
> ```
> Vale considerar tornar isso o padrão: trocar o fallback `?? "3000"` por uma
> porta improvável (ex.: 3457), já que 3000 é a porta que todo projeto Next usa
> — e é justamente por isso que ela colide.


| # | Item | Estado |
|---|---|---|
| 6 | Corrigir o claim do outbox + idempotência do consumidor (§1.1) | ✅ feito (claim); idempotência do consumidor pendente |
| 7 | Timeout por mensagem + reaper de `PROCESSING` preso (§1.4-1) | ⬜ |
| 8 | `WriteTimeout` do SSE + persistir resposta parcial (§1.4-2) | ⬜ |
| 9 | Rate limit por `user_id` em upload e chat (§1.4-3) | ⬜ |
| 10 | `Unscoped()` no delete, corte por runes, logar erros do worker (§1.4-5,6,7) | ⬜ |
| 11 | Gerar os `.dbml` do Apêndice A ou remover a promessa | ⬜ |

#### Correção do outbox — o que mudou

O claim agora reivindica de verdade, na mesma transação da seleção:

```go
Updates(map[string]any{
    "status":       domain.OutboxEventStatusInFlight,  // novo estado
    "attempts":     gorm.Expr("attempts + 1"),
    "available_at": now.Add(visibilityTimeout),        // rede de segurança
    "updated_at":   now,
})
```

`IN_FLIGHT` entra no filtro de seleção **de propósito**: se o `available_at`
venceu, o worker que reivindicou morreu antes de publicar, e o evento precisa
voltar para a fila sozinho — sem intervenção manual e sem reaper separado.

Arquivos: `domain/indexing.go` (constante), `repository/gorm_outbox_repository.go`
(claim), `processing/outbox_dispatcher.go` (visibility timeout + erros logados
em vez de engolidos), `config/config.go` (`OUTBOX_VISIBILITY_TIMEOUT_SECONDS`,
default 120s), `cmd/indexer/main.go` (wiring).

Duas decisões que vale registrar:

1. **`MarkEventPublished` falhando não aborta mais o lote.** Antes o `return`
   deixava os eventos seguintes — já reivindicados — presos até o timeout
   vencer. Agora loga e segue; o evento problemático volta pela mesma rede.
2. **A validação de config rejeita `visibility <= poll`.** Se a janela for menor
   que o intervalo de poll, o evento reaparece antes de o dispatcher terminar de
   publicar — que é exatamente a duplicação que o claim existe para evitar. O
   erro aparece no boot, não em produção.

**5 testes de integração novos, rodados contra Postgres 16 real** (container
descartável, não mock). Verifiquei que pegam o bug em três variantes:

| Código | Resultado |
|---|---|
| Original (só `attempts++`) | ✗ `evento devolvido deveria vir como IN_FLIGHT, veio "PENDING"` |
| Original + projeção em memória mentindo | ✗ `evento persistido deveria estar IN_FLIGHT, esta "PENDING"` |
| **Correção pela metade** (status sim, `available_at` não) | ✗ `available_at deveria ter avancado para o futuro` |
| Correção completa | ✓ 18/18 na suíte de integração inteira |

A terceira linha é a que importa: o teste não aceita meia correção. Quem mexer
nisso depois não consegue "consertar" só o status e achar que resolveu.



### Fase 2 — 2 semanas, instrumentação (sem instrumento não há dado)

| # | Item |
|---|---|
| 12 | `-coverprofile` no CI + badge — é número que a banca pede |
| 13 | TPC: checador de proveniência completa no benchmark |
| 14 | Determinismo: teste de 30 execuções com hash idêntico |
| 15 | Latência p50/p95 e bytes de `/collections/:id/context` × n |
| 16 | Promover o contador de consultas SQL de teste a middleware opcional |
| 17 | Separar custo **medido** de custo **estimado** no monitoramento |
| 18 | Curva de escala n ∈ {10, 50, 100, 500, 1000} — fecha o único ❌ dos RNFs |

Sobre o item 17: hoje `TotalEstimatedCostUSD` é sempre estimativa. Verifiquei que
`chatCompletionResponse` (`pkg/provider/factory.go:56-63`) parseia só `choices` e
`error` — **o campo `usage` da resposta da OpenAI, que traz a contagem exata de
tokens, é descartado**. E `estimateTextTokens` usa `runes/4`, enquanto
`estimateTokenCount` usa contagem de palavras: **duas heurísticas diferentes no
mesmo sistema**. Adicionar `usage` ao struct é meia hora de trabalho e troca
"estimamos" por "medimos" numa das três dimensões que o artigo promete avaliar.

### Fase 3 — 5 a 6 semanas, o trabalho

| # | Item |
|---|---|
| 19 | Corpus C1 expandido (30 docs, 150 consultas) com as 3 regras anti-circularidade |
| 20 | Corpus C2 real público PT-BR, com qrels graduados e κ reportado |
| 21 | pgvector no compose + braços B (BM25), C (denso), D (RRF) |
| 22 | `cmd/retrieval-eval` — 4 braços × 2 chunkings, emite `results.csv` |
| 23 | `scripts/analyze.py` — gera as tabelas em LaTeX direto do CSV, zero transcrição manual |
| 24 | Reescrever o artigo: tempo verbal, H1-H3, Ameaças à Validade, +12 referências |

### Fase 4 — paralelo, quando der

Fechar RFs parciais (logout, DELETE de categoria e coleção, rename), migrations
versionadas no lugar do `AutoMigrate`, `@Summary` nas rotas faltantes (~28 de 50),
repositório público com DOI no Zenodo, SUS com 8-12 participantes se quiser
sustentar o RNF01.

**Se só houver tempo para uma coisa:** Fase 3, itens 21-23. Um TCC com sistema
mediano e experimento rigoroso defende melhor que um TCC com sistema excelente e
zero comparação — que é exatamente a situação atual.

---

## 5. O que não precisa de trabalho

Tão importante quanto a lista de problemas.

- **Segurança de acesso.** Multi-tenancy, IDOR, SQL injection, path traversal,
  SSRF e criptografia foram auditados sem achado. O AES-GCM com chave derivada
  por usuário está correto e testado.
- **Higiene de build.** Build, vet, gofmt, lint, typecheck, testes unitários e
  build de produção: todos limpos. Não gaste tempo aqui.
- **Pipeline determinístico de indexação.** Recém-consertado, com teste de
  regressão por bug. Não remexer sem motivo novo.
- **Arquitetura relacional.** Coerente e sustenta a tese. Não há motivo técnico
  para trocar por grafo ou vetor dentro do escopo.
- **`AutoMigrate`.** As três listas conferem — 21 modelos em todas.
- **Custo de LLM por consulta de chat.** Verificado: é O(1), o prompt é truncado
  por construção. A tese de custo previsível se sustenta nesse eixo (§2.4).
- **Os controles `flat`/`hybrid` do benchmark.** São ablações reais e bem-feitas.
  Estão subaproveitados, não errados — viram B3a/B3b ao lado das baselines
  externas.

---

## 6. Nota de método

O spec-kit foi instalado (`.specify/` + `.claude/skills/speckit-*`) e a
constitution foi preenchida com os princípios reais do projeto, extraídos do
histórico de PRs e do `estado-do-projeto.md` — inclusive as lições aprendidas na
marra (o `.gitignore` que escondeu o middleware de JWT, o pre-commit que forçava
`--no-verify`, o CI que precisa instalar poppler e tesseract para não passar por
skip silencioso).

Para os próximos incrementos vale usar o fluxo: `/speckit-specify` para o
experimento comparativo, `/speckit-plan`, `/speckit-tasks`. A constitution já
trava as decisões que não podem ser negociadas por prazo — em especial
"auditabilidade sobre conveniência" e "honestidade nos números", que são
exatamente as que a pressão de fim de semestre costuma dobrar.

O maior risco do TCC II **não é o experimento dar resultado desfavorável**. Um
resultado do tipo *"empata em precisão estrutural, ganha por larga margem em
auditabilidade e determinismo, e perde em custo acima de M documentos"* é melhor
que vitória em todas as frentes: é crível, é específico, e delimita quando usar a
abordagem. O risco real é chegar na defesa com as mesmas três afirmações não
medidas que o TCC I já fazia.
