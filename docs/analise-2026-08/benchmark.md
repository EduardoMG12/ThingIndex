# Auditoria científica do benchmark do ThingIndex e plano para transformá-lo em evidência publicável

## 0. Veredito em uma frase

O benchmark atual é um **teste de regressão determinístico bem construído**, mas **não é um experimento científico**: ele mede casamento exato de strings contra um gabarito escrito pelo próprio autor sobre 5 documentos sintéticos (3 inline + 2 PDFs gerados pelo projeto), com N=8 consultas, sem repetições, sem variância, sem baseline externa e sem teste estatístico. Reportar `structure=100 / node_top1=100 / exact=100` nessas condições é, do ponto de vista da banca, **evidência de que o benchmark não discrimina**, não de que o sistema é bom.

---

## 1. O que cada métrica mede de fato (leitura do código)

Arquivo: `api/internal/usecase/indexing_benchmark.go` (1741 linhas). Fluxo em `GetPublicIndexingBenchmark` (l. 236–580).

### 1.1 `structure_pass_rate`

Calculada em `percentage(structurePasses, len(corpus.Documents))` (l. 510), onde `structurePassed` vem de `benchmarkDocumentStructurePassed` (l. 583–604):

```go
if len(result.TableSections) != fixture.ExpectedTableCount { return false }
if len(fixture.ExpectedOutlineTitles) == 0 { return true }
// para cada título esperado, exige presença exata (após TrimSpace) no conjunto de nós
```

**O que realmente mede:** um booleano por documento — "a contagem de tabelas bateu **exatamente** E todos os títulos esperados aparecem literalmente entre os títulos dos nós de outline". Denominador = 5 documentos.

**Fragilidades:**
- **Granularidade binária com N=5.** O espaço de valores possíveis é `{0, 20, 40, 60, 80, 100}`. Não é uma métrica contínua; é um contador de 5 bits. Qualquer variação é um salto de 20 pontos. Isso explica o `60 → 100` histórico: **3 acertos viraram 5 acertos**. Não há como calcular desvio-padrão sobre isso de forma honesta.
- **Assimetria não penalizada.** O teste é de *recall* puro sobre títulos: se o indexador produzir 40 nós lixo e os 4 esperados estiverem lá, passa. A precisão só aparece de forma decorativa em `unexpectedHeadingCount` (l. 639), que **não entra em nenhuma métrica de aprovação**.
- **Comparação por string literal.** `strings.TrimSpace(node.Title)` — sem normalização de acento, caixa, hífen ou numeração. Isso torna a métrica *hipersensível* a detalhes de renderização do PDF e *insensível* a qualidade semântica. É exatamente por isso que consertar a fixture do PDF (palavras sobrepostas) moveu a métrica: o pipeline não mudou, o **input** mudou.
- **`expected_table_count` é igualdade estrita.** Detectar 2 tabelas onde se esperava 1 reprova tanto quanto detectar 0. Não distingue *falso positivo* de *falso negativo*, que são erros com custos operacionais completamente diferentes.

### 1.2 `tree_node_top1_accuracy`

`percentage(nodeTop1Passes, totalDocumentQueries)` (l. 511). Um acerto é (l. 392):

```go
nodeMatched = tree.NodeHits[0].Title == query.ExpectedTopNodeTitle
```

**O que realmente mede:** para cada uma das 6 consultas de documento, se o **primeiro** nó retornado por `buildTreeRetrieval` tem título **byte-a-byte igual** ao esperado. É `Accuracy@1` — que na literatura de IR é `P@1` num cenário de relevância única, também interpretável como `Hit Rate@1`.

**Onde é circular ou frouxa:**
- **Circularidade do gabarito.** `expected_top_node_title` no `corpus.json` é uma string **idêntica** a um item de `expected_outline_titles` do mesmo documento (`"2 Payment Terms"`, `"3 DISPOSITIVO"`, `"3 BUSINESS CONTINUITY"`). Ou seja: o gabarito de recuperação é definido em função da **saída estrutural que o próprio pipeline produz**. Se o extrator renomear o nó para `"3. Dispositivo"`, o gabarito quebra — não porque a recuperação piorou, mas porque a *rotulação* mudou. Isso é **acoplamento gabarito↔implementação**, o defeito clássico que invalida um benchmark.
- **Consultas construídas a partir do texto-alvo.** `"payment deadline invoice receipt"` para um trecho que diz literalmente *"Invoices must be paid within 15 days of receipt"*. `"alternate region annual recovery exercise"` casa palavra por palavra com a seção. O scorer é `retrievalScore` (`document_usecase.go` l. 2136), que é **contagem de substrings**:

```go
for _, term := range queryTerms { if strings.Contains(normalized, term) { score++ } }
```

Ou seja, o teste é: *"um matcher de substring encontra o parágrafo cujas palavras eu copiei"*. Isso é **tautológico**. Não há sinônimo, não há paráfrase, não há termo ausente, não há distrator lexicalmente próximo. Um `grep` acerta 100%.
- **Um único ponto de decisão por documento.** 4 dos 5 documentos têm exatamente **1** consulta. A métrica tem denominador 6. Intervalo de confiança de Wilson para 6/6: **[61%, 100%]**. Reportar "100%" sem esse intervalo é sonegação de incerteza.
- **Sem casos negativos.** Não existe consulta cuja resposta correta seja "nenhum nó relevante". Portanto não se mede *falso positivo de recuperação* — o sistema nunca é punido por responder com confiança a algo que não existe no corpus.

### 1.3 `exact_match_accuracy`

`percentage(totalExactPasses, totalExactCases)` (l. 517), com `totalExactCases` somando consultas de documento (l. 379) **e** de coleção (l. 465), e `totalExactPasses` incrementado quando `nodeMatched && pageMatched` (l. 416) ou `documentMatched` (l. 486).

**O que realmente mede:** uma métrica **agregada heterogênea** — mistura 6 casos de "nó E página corretos no top-1" com 5 casos de "documento correto no top-1", num denominador único de 11.

**Fragilidades:**
- **Agregação de unidades incomparáveis.** A dificuldade de acertar (nó, página) num documento de 2 páginas é totalmente diferente de acertar o documento certo entre 5. Somar os dois num percentual único é indefensável metodologicamente. Uma banca vai perguntar: *"exact match de quê, exatamente?"*.
- **`pageMatched` é quase gratuito.** 3 dos 5 documentos são texto/HTML de página única (`expected_top_page_number: 1`); os 2 PDFs têm 2 páginas. A componente "página" acerta por construção na maioria dos casos.
- **Micro-corpus para coleção.** Escolher o documento certo entre **5**, com consultas que contêm o termo discriminante literal (`"bridge commander executive notification"` só aparece no playbook), tem baseline aleatória de 20% e baseline `grep` de ~100%. Não há evidência de escalabilidade discriminativa.

### 1.4 Sinais e controles que existem — e por que ainda não bastam

- `TreeNodeRecallAt3`, `TreePageRecallAt3`, `CollectionDocumentRecallAt3` (l. 512–516): já são `Recall@3`, mas com o mesmo denominador minúsculo, e no top-1=100% eles são **necessariamente** 100% — informação zero.
- **Controles flat/hybrid** (`buildFlatHybridControlTreeRetrieval`, `buildFlatPageControlTreeRetrieval`, l. 1028–1305): esta é **a parte cientificamente mais valiosa do arquivo** e está subaproveitada. São ablações reais (árvore vs. página plana vs. híbrido). Mas: (a) todos os controles usam **o mesmo scorer lexical** `retrievalScore`, então a comparação é "árvore vs. sem-árvore" e não "relacional vs. vetorial"; (b) com o corpus atual saturado em 100%, **os controles empatam**, e `summarizeIndexingBenchmarkRetrievalComparison` (l. 943) devolve "empate" — a ablação não produz sinal.
- `reference-workspace.json` compara contra o `pageindex/results` normalizado. O próprio código admite (l. 270): *"quality bar, not as perfect ground truth"*. Como baseline externa isso é fraco: é uma comparação de perfil estrutural, não de acerto de recuperação.
- `estimateIndexQualityScore` / `AverageHeuristicScore` (l. 332, 518): heurística de operação. Corretamente separada da acurácia no relatório (l. 257). **Não deve aparecer como resultado no TCC**, apenas como instrumento de monitoração — se aparecer, a banca vai chamar de métrica autoavaliativa.

### 1.5 O problema da fixture — como tratar isso no texto

`docs/estado-do-projeto.md` (l. 388–399) registra honestamente que parte do salto `60/66.7/81.8 → 100/100/100` veio de corrigir `supplier-master-order-form.pdf`, que renderizava palavras sobrepostas. Isso significa que **a métrica estava medindo um defeito do gerador de fixtures, não do pipeline**.

Isso não é vergonha — é **material de tese**, se apresentado como tal. A leitura correta: *"a sensibilidade da métrica a um artefato de renderização do corpus demonstra que a métrica é dominada pela qualidade da camada de extração de texto, não pela lógica de indexação"*. Isso justifica, cientificamente, tudo o que vem a seguir: separar erro de extração (OCR/parsing) de erro de estruturação de erro de recuperação, e medir cada camada isoladamente. O commit isolado (`3dac860`) é uma vantagem: permite reportar a tabela **antes/depois da correção da fixture** como evidência de sensibilidade.

---

## 2. Corpus: o que existe, por que é insuficiente, e como expandir legalmente

### 2.1 Diagnóstico do corpus atual

| Item | Valor atual | Problema |
|---|---|---|
| Documentos | 5 (3 texto/HTML inline, 2 PDF gerados) | Nenhum documento do mundo real |
| PDFs | 2, ~2 KB cada, 2 páginas | Não exercitam layout complexo, coluna dupla, nem OCR |
| Escaneados | 0 | O caminho de OCR **nunca é medido** |
| Tabelas | 1 (em HTML) | Extração de tabela em PDF: não medida |
| Páginas máx. | 2 | Escalabilidade intra-documento: não medida |
| Idioma | 4 EN, 1 PT | Viés; o domínio-alvo é PT-BR jurídico/administrativo |
| Consultas | 6 doc + 5 coleção = 11 | Insuficiente para qualquer teste estatístico |
| Distratores | ~0 | Nenhum par de documentos disputa a mesma consulta |

Regra prática de IR: para detectar uma diferença de ~10 p.p. em P@1 com poder estatístico razoável, precisa-se de **50–100 consultas**. Com 11, o erro-padrão é maior que qualquer efeito plausível.

### 2.2 Corpus proposto: 60 documentos, 5 estratos, ~180 consultas

Todos de **domínio público ou licença aberta**, obteníveis legalmente no Brasil. Meta: **~60 documentos**, o que é defensável para um TCC (comparável a corpora de avaliação de pipeline de documentos) e ainda executável num notebook.

| Estrato | Fonte | Qtd | Características | Por que entra |
|---|---|---|---|---|
| **A. Jurídico nativo-digital** | Acórdãos e decisões do STF/STJ (portal de jurisprudência, acesso público; art. 93, IX CF — publicidade), DJe | 15 | PT-BR, 2–30 pág., estrutura canônica (Relatório/Voto/Ementa/Dispositivo), sem tabela | Domínio-alvo; estrutura hierárquica forte e verificável |
| **B. Diário Oficial / atos administrativos** | DOU (in.gov.br, conteúdo público), diários oficiais estaduais/municipais | 10 | PT-BR, coluna dupla, texto denso, muitos atos por página, **alta densidade de distratores** | Testa segmentação e desambiguação entre documentos parecidos |
| **C. Editais e contratos públicos** | Portal de Compras Gov (comprasnet), editais de licitação, contratos publicados por Lei de Acesso à Informação (Lei 12.527/2011) | 12 | PT-BR, **muitas tabelas** (planilha de preços, cronograma), 20–120 pág. | Único estrato que estressa tabela + documento longo |
| **D. Documentos escaneados / OCR** | Acervos digitalizados públicos: BN Digital, Arquivo Nacional, Memória Legislativa; ou re-scan controlado de A/B (imprimir→digitalizar em 150/300 dpi) | 13 | Imagem, ruído, skew, qualidade variável | **O único caminho que hoje não tem medida nenhuma** |
| **E. Científico open access** | SciELO (CC-BY), periódicos DOAJ, arXiv (licenças permissivas) | 10 | PT/EN, layout 2 colunas, tabelas, figuras, referências | Padrão da literatura de extração estrutural; permite comparar com resultados publicados |

**Sobre normas ABNT:** as normas em si são **licenciadas e não redistribuíveis** — não coloque PDFs da ABNT no repositório. Use apenas normas técnicas de acesso livre (ex.: normas de agências reguladoras publicadas em DOU, resoluções ANVISA/ANEEL) que são atos administrativos públicos.

### 2.3 Estratificação cruzada (o que rende gráfico)

Cada documento recebe **rótulos ortogonais**, e é isso que permite reportar resultado **por fatia** — que é onde nasce o gráfico de barras agrupadas do artigo:

- `origem` ∈ {nativo-digital, escaneado}
- `tabelas` ∈ {0, 1–3, >3}
- `extensão` ∈ {curto ≤5 pág., médio 6–30, longo >30}
- `idioma` ∈ {pt, en}
- `estrutura` ∈ {numerada explícita, títulos sem numeração, sem hierarquia clara}

Alvo mínimo por célula relevante: 8–10 documentos, para que a fatia tenha IC utilizável.

### 2.4 Como montar o gabarito **sem circularidade** — protocolo obrigatório

Esta é a seção que salva o TCC na arguição. Três regras:

**Regra 1 — O gabarito estrutural é feito a partir do PDF, nunca da saída do sistema.**
Procedimento: um anotador abre o PDF (não o JSON do ThingIndex) e transcreve a hierarquia de seções como ela aparece **visualmente**, num esquema neutro:

```
{"doc_id": "...", "sections": [
  {"level": 1, "title_verbatim": "3. DISPOSITIVO", "start_page": 2, "end_page": 2}
]}
```

A comparação com a saída do sistema é feita depois, com **normalização declarada e simétrica** (lowercase, remoção de diacríticos via NFD, colapso de whitespace, remoção do prefixo de numeração) e **similaridade de Levenshtein normalizada ≥ 0.90** em vez de igualdade de string. Isso descola o gabarito da convenção de rotulação do parser. Documente a função de normalização no artigo — ela é parte do método.

**Regra 2 — As consultas são escritas a partir da *necessidade de informação*, não do texto-alvo.**
Proibido copiar palavras do trecho. Protocolo: o anotador lê o documento, formula a pergunta que um usuário real faria, **fecha o documento**, e só então escreve a query. Três variantes por necessidade de informação, para medir robustez lexical:
- `Q_literal`: usa termos do documento (o que existe hoje)
- `Q_paráfrase`: sinônimos/reformulação ("prazo para entrega dos documentos" → "quando preciso juntar as peças")
- `Q_ruidosa`: com erro de digitação e abreviação, como usuário real digita

Reportar as três separadamente é **o gráfico mais forte do artigo**: a queda de P@1 de literal→paráfrase→ruidosa mede exatamente o que o benchmark atual esconde. Um matcher de substring desaba em `Q_paráfrase`; se o ThingIndex também desabar, isso é resultado honesto e discutível — e aponta o trabalho futuro (expansão de consulta, embeddings) com evidência.

**Regra 3 — Dupla anotação e concordância inter-anotador.**
Pelo menos **20% do corpus** anotado por 2 pessoas independentes. Reportar **Cohen's κ** para os julgamentos binários de relevância e **acordo exato + Levenshtein médio** para os títulos de seção. κ ≥ 0.70 é o piso apresentável. Divergências resolvidas por adjudicação, com o protocolo descrito. Sem isso, "gabarito" é opinião do autor.

**Regra 4 — Distratores obrigatórios.** Para cada consulta, garantir que exista ao menos 1 documento *não-relevante* que compartilhe ≥3 termos com a consulta. Sem distrator, `P@1` não mede discriminação.

**Regra 5 — Casos negativos.** 10% das consultas devem ter resposta correta = "nenhum documento relevante". Mede taxa de falso positivo de recuperação — hoje inexistente.

**Congelamento e versionamento:** publicar `corpus-v1.0` com hash SHA-256 por arquivo, manifest com URL de origem + data de captura + licença. Guardar um **split de desenvolvimento (20%)** e um **split de teste (80%) que só é executado uma vez, no final**. Se o pipeline for ajustado olhando o resultado do teste, o número deixa de valer.

---

## 3. Métricas adicionais — o que a literatura de IR exige

### 3.1 Camada de recuperação (documento e nó)

Substituir a métrica única `exact_match` por um painel padrão, reportado **separadamente por escopo** (nó dentro do documento / documento dentro da coleção):

| Métrica | Definição | Por que ela entra | Onde discrimina |
|---|---|---|---|
| **P@1, P@3, P@5** | fração dos k primeiros que são relevantes | comparável com toda a literatura | separa "acertou por sorte no top-1" de "ranqueia bem" |
| **R@5, R@10** | fração dos relevantes recuperada nos k primeiros | mede cobertura; essencial quando há múltiplos relevantes | expõe corpus com distratores |
| **MRR** | média de 1/posição do primeiro relevante | contínua, sensível a mudanças de 1 posição | **substitui top-1 binário; é onde a variância aparece** |
| **nDCG@10** | ganho descontado, com relevância graduada (0/1/2/3) | padrão-ouro de ranking; usa relevância graduada | única métrica que aproveita "parcialmente relevante" |
| **MAP** | média das precisões médias | resumo global do ranking | bom para a tabela-resumo |
| **Success@k / taxa FP** | acerto binário e falso positivo nos casos negativos | mede excesso de confiança | novo eixo, hoje inexistente |

Relevância **graduada** (não binária): 0 = irrelevante, 1 = tangencial, 2 = relevante, 3 = responde diretamente. É isso que habilita nDCG e faz a métrica parar de saturar.

### 3.2 Camada de estruturação (outline)

| Métrica | Definição | Substitui |
|---|---|---|
| **Precision / Recall / F1 de seções** | casamento de seções detectadas × gabarito, com Levenshtein ≥0.90 e sobreposição de páginas | o booleano `structure_passed` |
| **Erro de fronteira de página** | |página_detectada − página_real| do início de cada seção, MAE e mediana | nada (hoje não existe) |
| **Acurácia de nível hierárquico** | fração de seções com o `level` correto | nada |
| **TEDS / TEDS-Struct** | Tree-Edit-Distance Similarity entre a árvore HTML da tabela extraída e a de referência (padrão do PubTabNet/ICDAR) | `expected_table_count == n` |
| **GriTS** (alternativa/complemento) | similaridade célula-a-célula em grade; decompõe em GriTS-Top (topologia), GriTS-Con (conteúdo), GriTS-Loc (posição) | idem |
| **Precision/Recall de tabelas detectadas @ IoU 0.5** | detecção antes de reconstrução | contagem estrita |

`TEDS` é contínuo em [0,1] — dá média ± desvio, dá boxplot, dá teste estatístico. `expected_table_count == n` dá um booleano. Essa é a diferença entre um gráfico e uma frase.

### 3.3 Camada de extração de texto / OCR

| Métrica | Definição | Uso |
|---|---|---|
| **CER** (Character Error Rate) | Levenshtein a nível de caractere / nº caracteres de referência | métrica principal para PT-BR (acento, cedilha) |
| **WER** (Word Error Rate) | (S+D+I)/N em palavras | comparável com literatura de OCR |
| **Cobertura de texto** | caracteres extraídos / caracteres de referência | detecta página que voltou vazia |
| **Taxa de acionamento de fallback OCR** | % de páginas que caíram no caminho OCR | conecta com o bug de "OCR inalcançável" já documentado |

Gabarito de OCR: para os 13 escaneados, transcrição manual de **2 páginas amostradas por documento** (26 páginas) — é trabalho viável e suficiente para IC. Para os re-scans controlados do estrato D, o texto nativo do documento original **é** o gabarito perfeito, custo zero. **Faça isso**: gerar escaneados por imprimir→digitalizar os documentos dos estratos A/B dá gabarito de OCR gratuito e perfeitamente alinhado, e ainda permite variar dpi (150/300/600) e ruído como **variável independente controlada** — o que rende a curva CER × dpi, um gráfico excelente e barato.

### 3.4 Baselines comparativas (sem elas nada disso significa nada)

Mínimo de 4 sistemas na tabela principal:

1. **B0 — Aleatório/oráculo trivial**: piso teórico. Custa 10 linhas de código e ancora todos os números.
2. **B1 — BM25 sobre texto plano** (Lucene/Tantivy/`bm25s` em Python, ou `ts_rank` do PostgreSQL). É a baseline lexical obrigatória. O `retrievalScore` atual é *pior que BM25* (contagem de substring sem IDF nem normalização de comprimento) — reportar isso é honesto e mostra domínio da área.
3. **B2 — Vetorial denso**: chunking fixo 512 tokens + embeddings (`multilingual-e5` ou `bge-m3`, ambos rodam local e grátis) + FAISS/pgvector, top-k cosseno. É a alternativa que o artigo promete comparar.
4. **B3 — ThingIndex relacional** (o sistema).
5. *(opcional, forte)* **B4 — Híbrido BM25 + denso com RRF** — o estado da prática atual; se o ThingIndex perder para ele, discuta *onde* ganha (custo, explicabilidade, agregação).

Os controles `flat`/`hybrid` já implementados em `indexing_benchmark.go` viram **ablações internas** (B3a: sem árvore; B3b: só páginas), complementares às baselines externas. Ablação responde "qual componente contribui"; baseline responde "vale a pena comparado ao que já existe". O artigo precisa das duas.

---

## 4. Protocolo experimental completo

### 4.1 Unidades e fatores

- **Unidade de observação de qualidade:** a consulta (n ≈ 180) e o documento (n = 60). Não a execução.
- **Unidade de observação de custo/latência:** a execução (repetições).
- **Fator principal:** sistema ∈ {B0, B1, B2, B3, B4} — **within-subjects**: todos os sistemas rodam sobre as **mesmas** consultas. Isso é crucial: permite testes pareados, que têm muito mais poder que testes independentes.
- **Fatores de bloco:** estrato do corpus, tipo de consulta (literal/paráfrase/ruidosa), origem (nativo/escaneado).

### 4.2 Repetições

| Tipo de medida | Repetições | Justificativa |
|---|---|---|
| Qualidade (P@k, MRR, nDCG, F1, TEDS, CER) com pipeline determinístico | **1 execução** + verificação de determinismo (3 execuções, exigir hash idêntico do output) | rodar 30× um pipeline determinístico não gera variância real — gera falsa precisão. A variância vem da **amostra de consultas**, e é capturada por bootstrap. |
| Qualidade com componente LLM (se houver prompt no caminho) | **5 execuções**, temperatura fixada e reportada | há estocasticidade real; reportar média ± dp entre execuções |
| Latência / CPU / memória | **30 execuções** por configuração, após **5 de warm-up descartadas** | distribuição assimétrica; precisa de p50/p95/p99 |
| Escalabilidade (curva N) | 10 repetições por ponto de N | suficiente para IC no gráfico |

**Determinismo é uma vantagem, não um problema** — só precisa ser declarado: *"o pipeline é determinístico (verificado por 3 execuções com hash de saída idêntico); portanto a incerteza reportada é a incerteza de amostragem sobre o conjunto de consultas, estimada por bootstrap"*.

### 4.3 Controle de variância

- Máquina dedicada, sem outras cargas; especificar CPU/RAM/SSD/SO/versão Go/versão PostgreSQL.
- `GOMAXPROCS` fixo; governor de CPU em `performance`; turbo boost desabilitado se possível.
- Ordem de execução dos sistemas **aleatorizada** e intercalada (não rodar B1 30× depois B3 30× — thermal drift vira efeito de sistema).
- Banco recriado do zero por execução (`VACUUM ANALYZE` antes de medir); estados de cache declarados: medir **cold** (cache derrubado) e **warm** separadamente.
- Seeds fixadas e publicadas.

### 4.4 Testes estatísticos

| Comparação | Teste | Por quê |
|---|---|---|
| P@1 / Success@k entre 2 sistemas, mesmas consultas | **McNemar** (com correção de continuidade) | pareado, resultado binário — é o teste correto e é o que a banca espera |
| MRR / nDCG / TEDS / CER entre 2 sistemas | **Wilcoxon signed-rank** (pareado, não-paramétrico) | as distribuições não são normais; não use t-test em nDCG |
| Alternativa robusta e cada vez mais padrão em IR | **Teste de permutação pareado (randomization test)**, 10.000 permutações | não assume nada sobre a distribuição; é o recomendado pela literatura de avaliação de IR |
| ≥3 sistemas sobre as mesmas consultas | **Friedman** + post-hoc **Nemenyi**, ou permutação com correção | evita inflação de erro tipo I |
| Múltiplas comparações | **Holm-Bonferroni** (ou Benjamini-Hochberg se muitas métricas) | obrigatório; reportar α ajustado |
| Latência entre sistemas | **Mann-Whitney U** em p50; reportar p95/p99 como descritivo | latência não é normal |

**Tamanho de efeito é obrigatório**, não opcional: `Cliff's delta` (não-paramétrico) ou `Cohen's d` para as contínuas, e diferença absoluta em p.p. com IC para as binárias. *"p < 0.05"* sozinho não convence banca nenhuma hoje.

**Intervalos de confiança:**
- Métricas binárias (P@1, taxa de acerto estrutural): **IC de Wilson 95%** — nunca a aproximação normal, que com n pequeno e p próximo de 1 dá limite superior >100%.
- Métricas contínuas (MRR, nDCG, TEDS, CER, latência): **bootstrap percentílico, 10.000 reamostragens** sobre as consultas/documentos.
- Diferenças entre sistemas: **IC bootstrap da diferença pareada** — se cruza zero, não há efeito, independente do p-valor.

### 4.5 Como reportar (esqueleto das tabelas do artigo)

**Tabela 1 — Caracterização do corpus** (contagens por estrato × origem × tabelas × extensão; total de páginas, total de consultas, κ inter-anotador).

**Tabela 2 — Qualidade de recuperação, coleção** (linhas = sistemas, colunas = métricas):

| Sistema | P@1 [IC95] | MRR [IC95] | nDCG@10 [IC95] | R@5 [IC95] | Taxa FP |
|---|---|---|---|---|---|
| B0 aleatório | 0.02 [0.00–0.05] | ... | ... | ... | ... |
| B1 BM25 | ... | ... | ... | ... | ... |
| B2 denso (e5) | ... | ... | ... | ... | ... |
| B3 ThingIndex | ... | ... | ... | ... | ... |
| B4 híbrido RRF | ... | ... | ... | ... | ... |

Com nota de rodapé: *"† diferença significativa vs. B1 (permutação pareada, p<0.01 após Holm; Cliff's δ = 0.31, efeito médio)"*.

**Tabela 3 — Qualidade por tipo de consulta** (literal / paráfrase / ruidosa) × sistema. **É aqui que o resultado interessante mora.**

**Tabela 4 — Extração estrutural**: F1 de seções, acurácia de nível, MAE de fronteira de página, TEDS de tabelas — por estrato.

**Tabela 5 — OCR**: CER/WER por dpi e por estrato; taxa de acionamento de fallback.

**Tabela 6 — Custo** (seção 5).

**Tabela 7 — Ablação interna**: ThingIndex completo vs. sem-árvore (flat) vs. só-páginas — reaproveitando os controles já codificados.

**Gráficos que valem a pena:**
1. Barras agrupadas: nDCG@10 por sistema × tipo de consulta, com barras de erro (IC95). *O gráfico principal do artigo.*
2. Curva P@k / R@k (k = 1..10) por sistema.
3. Boxplot de TEDS por estrato — mostra distribuição e outliers, impossível com métrica binária.
4. **Curva latência × N documentos** (log-log) — seção 6.
5. Scatter custo (USD ou ms) × qualidade (nDCG): a fronteira de Pareto. **Este é o gráfico que sustenta a tese**, porque o argumento do ThingIndex não é "sou o mais preciso", é "sou competitivo a custo muito menor".
6. CER × dpi para o estrato escaneado.
7. Barras antes/depois da correção da fixture — honestidade metodológica vira ponto positivo.

---

## 5. Medição defensável de custo operacional

O artigo promete custo; hoje não há **nenhuma** instrumentação de custo no benchmark. Proposta:

### 5.1 Dimensões e como instrumentar

| Dimensão | Unidade | Como medir (Go) |
|---|---|---|
| Tempo de indexação | ms/documento e **ms/página** (normalizado) | `time.Since` por etapa: parse → OCR → estruturação → chunking → persistência. Reportar o breakdown por etapa (gráfico de barras empilhadas). |
| CPU | segundos de CPU (user+sys) | `syscall.Getrusage(RUSAGE_SELF)` antes/depois, ou `/usr/bin/time -v`; para o processo inteiro, `cgroup v2` `cpu.stat` |
| Memória de pico | MB (RSS) | `ru_maxrss` do `Getrusage`; complementar com `runtime.ReadMemStats` (`HeapAlloc`, `TotalAlloc`) |
| Disco | **bytes/página** e bytes/documento, decomposto | `pg_total_relation_size()` por tabela antes/depois da ingestão do lote; separar dados de índices (`pg_indexes_size`) |
| Tokens LLM | tokens in/out por documento | contador no cliente do LLM; se não houver LLM no caminho de indexação, **dizer isso explicitamente — é um resultado, não uma lacuna**: custo marginal de LLM = 0 na indexação |
| USD | USD/1000 páginas | tokens × preço tabelado (citar tabela e data) + custo de compute (preço de instância cloud equivalente × horas de CPU) |
| Latência de consulta | ms, p50/p95/p99 | medir só o caminho de recuperação, cache quente e frio separados |

**Normalize por página, não por documento.** Documento é unidade de tamanho arbitrário; comparar "USD/documento" entre um acórdão de 3 páginas e um edital de 120 é ruído puro. Use **USD por 1.000 páginas** e **MB por 1.000 páginas** — números que um gestor entende e que a banca não consegue atacar.

### 5.2 Comparação justa com a alternativa vetorial

Para ser defensável, a comparação com B2 (denso) precisa igualar o que é comparável e separar o que não é:

**Custo de indexação (one-time):**
- ThingIndex: parse + estruturação + escrita relacional. Zero chamadas de embedding.
- Vetorial: parse (mesmo custo) + **embeddings de todos os chunks**. Este é o item dominante e é o argumento central. Reporte em duas modalidades, porque a banca vai perguntar por ambas:
  - **API paga** (ex.: `text-embedding-3-small`): tokens × preço tabelado → USD/1000 páginas.
  - **Local** (e5/bge-m3 em CPU e em GPU): segundos de compute × preço de instância equivalente.

**Custo de armazenamento:**
- ThingIndex: tabelas + índices B-tree/GIN.
- Vetorial: texto + **vetores** (`dim × 4 bytes × nº chunks`) + índice HNSW/IVF (overhead de 1.5–3× sobre os vetores). Com `bge-m3` (1024 dim) e chunks de 512 tokens, isso é dezenas de MB por milhar de páginas. Tabela com MB/1000 páginas para ambos.

**Custo de consulta:**
- ThingIndex: N consultas SQL (ver seção 6).
- Vetorial: 1 embedding da query (tokens + latência de API, ou compute local) + busca ANN. **Não esqueça o custo do embedding da query** — é recorrente e por consulta; é onde a diferença acumula em produção.

**Custo de atualização (o argumento mais forte e o mais esquecido):**
- Documento alterado no ThingIndex: reindexação incremental relacional.
- Vetorial: re-embedding dos chunks afetados + reconstrução parcial do índice ANN. Meça **custo de atualização de 10% do corpus** — em sistemas reais isso domina o TCO e favorece a abordagem relacional.

**Regra de honestidade:** rode B2 **no mesmo hardware**, com os mesmos documentos, e publique a configuração exata (modelo, dimensão, tamanho de chunk, overlap, tipo de índice, parâmetros HNSW). Comparar seu sistema local contra uma API remota sem descontar rede é o erro que a banca detecta na hora.

**Gráfico-alvo:** eixo X = custo total de propriedade (USD/1000 páginas, escala log), eixo Y = nDCG@10, um ponto por sistema com barras de erro nos dois eixos. Se o ThingIndex ficar na fronteira de Pareto — mesmo perdendo em qualidade absoluta para o denso — **a tese está defendida com evidência**.

---

## 6. O teste de "agregação linear": o que ele prova e o que não prova

### 6.1 O que o teste faz (`gorm_collection_context_repository_integration_test.go`, l. 242–296)

`TestCollectionContextQueryCountDoesNotGrowWithDocuments` conta callbacks de query do GORM em `GetCollectionContext` para uma coleção com **1** documento e com **20**, e assere: (a) contagens iguais; (b) contagem == 9.

### 6.2 O que ele **de fato** prova

Prova, de forma limpa e verificável, **a ausência de N+1 no número de round-trips ao banco** para essa operação, em dois pontos amostrais. É um bom teste de arquitetura e merece estar no artigo. O comentário no código é honesto e o *budget* constante é uma boa prática.

### 6.3 O que ele **não** prova — e a banca vai perguntar exatamente isso

1. **Número de consultas ≠ custo.** Nove `SELECT ... WHERE document_id IN (...)` com 20 documentos custam mais que nove com 1 documento. O trabalho no servidor, o volume transferido e o tempo de resposta **crescem** com N. "Agregação linear" nomeada como "custo não cresce" é uma **afirmação forte demais** para a evidência apresentada. O nome correto do que foi provado é: **"o número de round-trips é O(1) em relação ao número de documentos"**. Corrija a redação no artigo — isso sozinho já evita uma pergunta desconfortável.
2. **Dois pontos não definem uma curva.** 1 e 20 são consistentes com O(1), mas também com qualquer função que só cresça depois de 20 (ex.: batching que estoure e faça chunking de `IN`, limite de parâmetros do driver, spill de memória).
3. **N=20 é minúsculo.** Nenhuma degradação de plano de execução aparece nessa escala. O PostgreSQL pode trocar `Index Scan` por `Seq Scan` a partir de certa seletividade — e aí o comportamento muda qualitativamente.
4. **Não mede o que o usuário sente.** Não há latência, memória, nem bytes transferidos.
5. **Ambiente de teste ≠ produção.** Se `openIntegrationDB` usa SQLite ou um Postgres sem dados representativos, o plano de execução não é o de produção.

### 6.4 Como fortalecer — o experimento de escalabilidade

Mantenha o teste unitário como *guard-rail* de regressão (ele é barato e pega N+1 no CI) e **adicione um experimento separado**:

**Desenho:** N ∈ {1, 10, 100, 1.000, 10.000} documentos (adicione 10k se o hardware permitir — é o ponto que convence), 10 repetições por ponto, ordem aleatorizada, warm-up descartado, medindo:

| Métrica | O que revela |
|---|---|
| Nº de consultas SQL | confirma O(1) em toda a faixa — a afirmação original, agora sustentada |
| Latência wall-clock p50/p95/p99 | o que o usuário sente |
| Bytes transferidos do banco | cresce linearmente; é o custo real |
| Linhas retornadas por consulta | liga latência a volume |
| Pico de RSS do processo | expõe o risco de materializar tudo em memória (o verdadeiro limite da abordagem) |
| Tempo de planejamento vs. execução (`EXPLAIN (ANALYZE, BUFFERS)`) | detecta mudança de plano; anexar o plano em N=100 e N=10.000 no apêndice |

**Contraste obrigatório:** implemente uma versão **N+1 deliberada** (um `SELECT` por documento) e plote as duas curvas juntas. Sem o contraste, "9 consultas" é um número sem escala; **com** o contraste, o gráfico mostra 9 consultas constantes contra 6N+3 crescendo — e a contribuição arquitetural fica visualmente óbvia. Este é provavelmente **o gráfico mais barato de produzir e mais persuasivo do TCC inteiro**.

**Análise:** ajustar `latência = a + b·N` e `latência = a·N^b` por mínimos quadrados; reportar R² de ambos e o expoente `b` com IC. Se `b ≈ 1` para a latência e o número de consultas permanece 9, a afirmação correta e defensável é: **"round-trips O(1), volume de dados O(N), latência empiricamente linear com coeficiente angular b = X ms/documento [IC95]"**. Isso é preciso, verificável e imune a contestação.

**Gráfico:** log-log, latência × N, duas séries (relacional agregado vs. N+1), barras de erro em cada ponto, eixo secundário com o número de consultas. Uma figura, três afirmações provadas.

---

## 7. Priorização — o que fazer com tempo limitado

Ordenado por (impacto na banca) ÷ (esforço):

| # | Ação | Esforço | Impacto |
|---|---|---|---|
| 1 | Curva latência × N com contraste N+1 (seção 6.4) | **baixo** — infra de teste já existe | **altíssimo**: 1 gráfico, 3 afirmações provadas |
| 2 | Trocar `retrievalScore` por BM25 como **baseline B1** e reportar ThingIndex vs. BM25 | baixo (`ts_rank` do Postgres já disponível) | alto: baseline obrigatória, resolve a crítica "comparado com o quê?" |
| 3 | Consultas em 3 variantes (literal/paráfrase/ruidosa) sobre o corpus **atual** | baixo | alto: quebra o teto de 100% imediatamente, cria variância |
| 4 | Trocar `structure_passed` binário por P/R/F1 de seções com Levenshtein | médio | alto: métrica contínua, elimina a circularidade de string exata |
| 5 | Instrumentar custo (tempo/etapa, CPU, RSS, bytes/página) | médio | alto: o artigo **promete** isso e hoje não entrega nada |
| 6 | MRR + nDCG@10 + IC bootstrap + teste de permutação | médio | alto: profissionaliza o painel de métricas |
| 7 | Expandir corpus para ~60 documentos com estratos e dupla anotação | **alto** | altíssimo, mas é o item que pode não caber no cronograma |
| 8 | Estrato escaneado por re-scan controlado (gabarito grátis) + CER × dpi | médio | alto: cobre o caminho de OCR, hoje com **zero** medição |
| 9 | Baseline vetorial (e5/bge-m3 + pgvector) + gráfico custo × qualidade | alto | altíssimo: é a comparação central da narrativa do artigo |

Se só houver tempo para três: **#1, #3 e #2**. Eles quebram a saturação em 100%, criam a baseline comparativa e produzem dois gráficos publicáveis, sem depender da expansão do corpus.

---

## 8. Como escrever a limitação sem se sabotar

Não esconda o corpus pequeno nem o episódio da fixture. Escreva algo na linha de:

> A avaliação inicial empregou um corpus controlado de 5 documentos e 11 consultas, adequado como teste de regressão determinístico do pipeline, mas insuficiente como evidência de acurácia generalizável: o pequeno número de casos e a construção lexicalmente literal das consultas produzem saturação (100% em todas as métricas), o que impede qualquer discriminação entre variantes. Adicionalmente, observou-se que parte da variação histórica das métricas (60% → 100% em conformidade estrutural) decorreu da correção de um artefato de renderização em uma fixture do corpus, e não de alteração no pipeline — evidência direta de que as métricas empregadas eram dominadas pela qualidade da camada de extração de texto. Essas duas constatações motivaram o redesenho experimental descrito na Seção X.

Isso converte a fragilidade em **justificativa metodológica**. Uma banca que encontra sozinha esse problema reprova; uma banca que lê o autor identificando e corrigindo o problema aprova.
