# Experimento comparativo de recuperação — metodologia e resultados

> **Estado: implementado e executável.** Tudo aqui é reproduzível com dois
> comandos. Nenhum número deste documento foi transcrito à mão.
>
> ```bash
> cd api && go run ./cmd/retrieval-eval -out ../docs/article/results/results.csv -summary
> python3 scripts/analyze.py docs/article/results/results.csv --outdir docs/article/tables
> ```
>
> **Braços densos (C e E):** os embeddings são gerados uma vez e versionados.
> Regerar só é necessário quando o corpus muda — e um teste falha se você
> esquecer.
>
> ```bash
> python3 scripts/embed-corpus.py \
>   --corpus api/internal/evalcorpus/testdata/corpus-c1.json \
>   --out    api/internal/evalcorpus/testdata/embeddings-c1.json
> cd api && go run ./cmd/dense-determinism -tex ../docs/article/tables/dense-determinism.tex
> ```

---

## 1. O problema que este experimento resolve

O artigo afirma, em quatro lugares, superioridade em auditabilidade,
determinismo e custo sobre RAG vetorial — e **nunca compara com nada**. O único
"comparativo" existente (`flat-page-control`, `flat-hybrid-control`) compara o
sistema com variações dele mesmo.

Pior: o benchmark antigo reportava `structure=100 / node_top1=100 / exact=100`.
Isso **não é evidência de que o sistema é bom** — é evidência de que a métrica
não discrimina. Três defeitos a tornavam tautológica:

| Defeito | Consequência |
|---|---|
| Gabarito circular (6 de 6 `expected_top_node_title` são cópia de `expected_outline_titles`) | Renomear `"3 DISPOSITIVO"` para `"3. Dispositivo"` quebra a métrica sem que a recuperação tenha mudado |
| Consultas copiando palavras do trecho-alvo | O teste vira *"um matcher de substring encontra o parágrafo cujas palavras eu copiei"*. Um `grep` acerta 100% |
| Zero casos negativos | Um sistema que sempre responde algo pontua igual a um que sabe dizer "não encontrei" |

---

## 2. O que foi construído

### 2.1 Braços comparados

| Braço | O quê | Onde |
|---|---|---|
| **A — relacional** | Scorer atual do ThingIndex: contagem de substring, sem IDF | `internal/retrieval/lexical.go` |
| **B — BM25** | Robertson & Zaragoza (2009), `k1=1.2`, `b=0.75` | `internal/retrieval/lexical.go` |
| **C — denso** | `multilingual-e5-base` (768d), varredura exata por cosseno | `internal/retrieval/dense.go` |
| **C′ — denso-IVF** | O mesmo modelo sob índice **aproximado** IVF-Flat, o do pgvector | `internal/retrieval/ivf.go` |
| **D — RRF** | Fusão de A e B por Reciprocal Rank Fusion, `k=60` (Cormack et al., 2009) | `internal/retrieval/metrics.go` |
| **E — RRF denso** | Fusão de B e C — o híbrido do estado da prática | `cmd/retrieval-eval/main.go` |

O braço A é **reprodução fiel** do scorer de produção, defeitos inclusive. Isso
é o ponto: o experimento mede o sistema como ele é, não uma versão idealizada.

### 2.1.1 Como o braço denso roda sem quebrar a hermeticidade

Uma versão anterior deste documento declarava o braço C como não implementável,
porque carregar um modelo de 1,1 GB tiraria a execução do regime hermético e
reproduzível em CI. **O diagnóstico estava certo; a conclusão, não.**

A solução é a que a literatura de replicabilidade em RI usa: separar geração de
busca. Os embeddings são computados **uma vez**, por `scripts/embed-corpus.py`,
e versionados como artefato JSON. O harness em Go só faz aritmética de ponto
flutuante sobre vetores lidos do arquivo.

Três consequências, todas desejáveis:

- o harness continua rodando em CI sem GPU, sem torch e sem rede;
- a latência medida deixa de estar contaminada por tempo de inferência;
- o resultado deixa de depender da versão de `transformers` da máquina.

**O risco que isso cria é o artefato dessincronizar do corpus** — e ele é fatal
justamente por ser silencioso: nada quebra, os números apenas passam a
descrever uma versão anterior do corpus. Dois guards independentes fecham a
brecha, ambos por hash SHA-256 do texto enviado ao modelo:

| Guard | Onde | Comportamento |
|---|---|---|
| Teste de CI | `internal/evalcorpus/embeddings_test.go` | falha o build, aponta o item divergente e imprime o comando de regeração |
| Guard de execução | `checkDenseSync` em `cmd/retrieval-eval` | **aborta a medição** antes de escrever qualquer número |

Verificados em RED→GREEN: editar uma consulta do corpus faz os dois falharem
identificando exatamente `q01-literal` / `q11-para`.

### 2.1.2 Por que duas variantes de denso, e por que isso importa

Medir apenas o denso **exato** produziria uma comparação enviesada **a favor da
tese deste trabalho**. A afirmação "o denso também é determinístico" é
verdadeira na varredura exata e falsa no índice aproximado — e é o aproximado
que se usa em escala, porque a varredura exata é O(N) por consulta. A premissa
de escalabilidade do RAG vetorial repousa inteiramente sobre IVF-Flat e HNSW.

O braço C′ reimplementa o IVF-Flat com os nomes e defaults do pgvector
(`lists = sqrt(N)`, `probes = 1`), para que o leitor consiga mapear o que foi
medido aqui no que ele configura em produção.

### 2.1.3 O piso de similaridade: uma hipótese que a medição refutou

A primeira implementação do braço denso usava um piso de similaridade de 0,80,
sob a hipótese razoável de que um limiar separaria relevantes de irrelevantes e
permitiria ao braço responder "nada encontrado" nos casos negativos.

**A medição refutou a hipótese** (`scripts/dense-floor-diagnostic.py`):

| Grupo | n | mín | mediana | máx |
|---|---|---|---|---|
| Pares relevantes | 18 | 0,7852 | 0,8335 | 0,8810 |
| Pares irrelevantes | 62 | 0,7626 | 0,8029 | **0,8483** |

As distribuições **se sobrepõem em quase toda a extensão**. Não existe limiar
que preserve os relevantes e corte os irrelevantes: em 0,80 o corte já descarta
2 relevantes enquanto 32 irrelevantes sobrevivem; em 0,84 restam 6 de 18
relevantes.

A causa tem nome na literatura: **anisotropia** do espaço de embeddings.
Modelos treinados com objetivo contrastivo concentram os vetores num cone
estreito, de modo que o cosseno tem piso alto mesmo entre textos sem relação. A
similaridade é útil como **ordem**, não como **magnitude**.

O piso foi zerado, e a decisão está travada por teste. Manter 0,80 teria dois
efeitos, ambos inaceitáveis: **prejudicaria a baseline concorrente** (vencer um
espantalho construído por mim) e **esconderia um achado** — a incapacidade de
dizer "não encontrei" atinge todos os braços, e agora sabemos que por razões
*diferentes*.

### 2.2 Corpus C1

`api/internal/evalcorpus/testdata/corpus-c1.json` — 5 documentos em PT-BR
(jurídico, licitação, contratual, normativo, operacional), 18 consultas.

As três regras anti-circularidade estão aplicadas e **travadas por teste**:

1. **Gabarito por `document_id`**, nunca por título produzido pelo pipeline. Se
   o extrator mudar a rotulação, a métrica não se move.
2. **Consultas escritas a partir da necessidade de informação.** As marcadas como
   `literal` copiam termos do alvo de propósito — são o **controle de teto**.
   `paraphrase` e `noisy` não podem ser literais na prática, e
   `CheckCircularity` falha o build se forem.
3. **Relevância graduada 0–3** e **casos negativos explícitos** (`negative:true`),
   distinguindo "ninguém anotou" de "nada é relevante".

### 2.3 Métricas

Painel padrão de RI, com as definições da literatura (Manning et al. cap. 8;
Järvelin & Kekäläinen 2002): **P@1/@3/@5, R@5/@10, MRR, nDCG@10** com ganho
exponencial `(2^rel − 1)/log2(i+2)`.

Cada métrica tem teste com valor **calculado à mão** a partir da definição
publicada — nunca a saída da própria implementação. Uma métrica errada não
quebra nada: devolve um número plausível e invalida o experimento em silêncio.

### 2.4 Estatística

- **Wilcoxon signed-rank pareado**, não teste t: métricas de RI são limitadas em
  [0,1] e não seguem normal.
- **Bonferroni** sobre todas as comparações.
- **Cliff's delta** com limiares de Romano et al. (2006) — *p*-valor diz se a
  diferença existe, não se ela importa.
- **IC 95% por bootstrap sobre as consultas**, semente fixa.

> A incerteza reportada é de **amostragem sobre o conjunto de consultas**, e não
> de variabilidade de execução. O pipeline é determinístico: rodar 30× o mesmo
> código e obter 30 resultados idênticos não gera variância, e fingir que gera
> seria desonesto.

A implementação do Wilcoxon é própria (para o script rodar sem dependência
científica) e foi **validada contra `scipy.stats.wilcoxon`**: maior divergência
de `1.4e-16` em 8 amostras aleatórias com *n* entre 12 e 40.

---

## 3. Resultados da execução atual

Corpus C1: 5 documentos, 18 consultas (2 negativas), 18 pares julgados.

| Braço | P@1 | MRR | nDCG@10 | Falso+ (negativas) |
|---|---|---|---|---|
| A — relacional | 0,750 | 0,830 | **0,869** | 1,000 |
| B — BM25 | 0,812 | 0,862 | 0,894 | 1,000 |
| C — denso (exato) | 0,688 | 0,823 | **0,844** | 1,000 |
| C′ — denso IVF | 0,688 | 0,781 | 0,797 | 1,000 |
| D — RRF (A+B) | 0,812 | 0,870 | **0,900** | 1,000 |
| E — RRF denso (B+C) | 0,812 | **0,880** | 0,890 | 1,000 |

### P@1 por variante de consulta

| Braço | Literal | Paráfrase | Ruidosa | Queda literal→paráfrase |
|---|---|---|---|---|
| A — relacional | 1,000 | 0,571 | 0,750 | **−0,429** |
| B — BM25 | 1,000 | 0,714 | 0,750 | −0,286 |
| C — denso | 0,600 | 0,571 | 1,000 | **−0,029** |
| C′ — denso IVF | 0,600 | 0,714 | 0,750 | **+0,114** |
| D — RRF | 1,000 | 0,571 | 1,000 | −0,429 |
| E — RRF denso | 0,800 | 0,714 | 1,000 | −0,086 |

### Estabilidade do ranking sob fatores que não são a pergunta

`go run ./cmd/dense-determinism` — 30 repetições, comparação por hash SHA-256
do ranking completo (identificador + score).

| Fator variado | Consultas estáveis | Máx. rankings distintos | Perda de recall |
|---|---|---|---|
| **C — denso exato**, 30 execuções | **100%** | 1 | — |
| C′ — IVF, 30 seeds | **0%** | 4 | **48,4%** |
| C′ — IVF, 30 ordens de inserção (seed fixa) | **0%** | 4 | — |
| C′ — IVF, `probes` de 1 a `lists` | **0%** | 2 | — |

---

## 4. Leitura honesta dos resultados

**Achado 1 — a queda literal→paráfrase é o resultado central, e o braço denso
mostra por quê.** Os braços lexicais acertam 100% quando a consulta copia
termos do alvo e despencam para 57–71% quando a mesma necessidade é expressa
com outras palavras. Isso quantifica exatamente o que o benchmark antigo
escondia: os 100/100/100 mediam casamento lexical, não recuperação. A queda é
maior no relacional (−0,429) do que no BM25 (−0,286), coerente com a ausência
de IDF.

O braço denso **inverte o padrão**: 0,600 em literal e 0,571 em paráfrase — ele
é o único cuja precisão praticamente não depende de a consulta reusar as
palavras do documento (queda de −0,029). É a evidência mais direta de que os
dois paradigmas falham por razões diferentes, e a justificativa empírica para
os braços híbridos.

**Achado 2 — o relacional não é inferior ao denso neste corpus.** nDCG@10 de
0,869 (A) contra 0,844 (C), com IC 95% amplamente sobrepostos. É exatamente a
forma de H1, e o resultado é favorável — **com a ressalva do achado 4**, que é
o que impede tratá-lo como demonstração.

**Achado 3 — nenhum braço reconhece a ausência de resposta, e agora sabemos por
quê.** Falso positivo 1,000 nos seis. Nos lexicais, porque qualquer termo em
comum gera score positivo. No denso, por uma razão mais interessante e mais
difícil de corrigir: a análise de separabilidade (§2.1.3) mostra que as
similaridades de pares relevantes e irrelevantes **se sobrepõem**, então nenhum
limiar resolve. Não é ajuste de parâmetro — é geometria do espaço de
embeddings.

**Achado 4 — as diferenças entre braços NÃO são estatisticamente
significativas.** Todos os *p* corrigidos dão 1,000 (75 comparações,
Bonferroni) e quase todos os Cliff's δ são desprezíveis. **Isso é limitação de
poder amostral, não evidência de empate.** Com 16 consultas positivas, o
Wilcoxon não tem como detectar efeito. O script devolve `p = 1.0` explicitamente
quando *n* < 10 em vez de fabricar um número que parece preciso.

**Achado 5 — o índice aproximado troca determinismo por escala, e o preço é
mensurável.** Este é o resultado que sustenta a contribuição do trabalho, e ele
é qualitativamente diferente dos anteriores:

- a varredura exata é **100% estável** em 30 execuções — é o controle, e ele
  isola a causa: a variação observada no IVF não vem do modelo nem da
  aritmética de ponto flutuante;
- trocar **apenas a seed** do k-means, sem tocar em corpus, modelo ou consulta,
  produz até **4 rankings distintos para a mesma pergunta**, com perda média de
  **48,4% de recall** em relação à varredura exata;
- trocar **apenas a ordem de inserção** dos documentos, com a seed fixa, produz
  o mesmo efeito. Este é o fator mais grave na prática, porque ordem de
  inserção não é configuração: é histórico de uso. **Dois usuários que subiram
  os mesmos documentos em ordem diferente podem receber respostas diferentes
  para a mesma pergunta**, e nenhum dos dois tem como perceber.

> **A afirmação que o artigo pode fazer agora é mais específica e mais
> defensável do que a original.** Não "somos determinísticos e o RAG vetorial
> não é" — que é falso para o denso exato — mas:
>
> *o denso exato é determinístico mas custa O(N) por consulta; o denso
> aproximado escala mas torna o resultado dependente de decisões de indexação
> que o usuário não controla nem observa. A recuperação relacional não paga
> nenhum dos dois preços.*

---

## 5. O que fica pendente

| Item | Por quê |
|---|---|
| Corpus para ~150 consultas, ≥30 documentos | Sem isso não há poder estatístico (§4, achado 4). **É o gargalo único** |
| Dupla anotação em ≥20% com Cohen's κ reportado | Sem κ, "gabarito" é opinião do autor |
| Repetir o experimento de determinismo com N maior | Com 5 documentos, `lists = sqrt(N) = 2`: o IVF tem pouco espaço para particionar, e o efeito medido pode estar amplificado |
| Braço denso com HNSW além de IVF-Flat | HNSW é o default do pgvector desde a 0.5.0 e tem fontes de não-determinismo próprias (`ef_construction`, ordem de inserção nos grafos) |
| Documentos escaneados (curva CER × dpi) | O caminho de OCR nunca foi medido |

---

## 6. Arquivos

```
api/internal/retrieval/metrics.go       métricas de RI + RRF
api/internal/retrieval/lexical.go       braços A (substring) e B (BM25)
api/internal/retrieval/dense.go         braço C: artefato, validação, busca exata
api/internal/retrieval/ivf.go           braço C′: IVF-Flat (k-means + probes)
api/internal/evalcorpus/corpus.go       carga, validação e checagem de circularidade
api/internal/evalcorpus/embeddings_test.go  guard de sincronia corpus × embeddings
api/internal/evalcorpus/testdata/       corpus C1 + embeddings versionados
api/cmd/retrieval-eval/main.go          harness: roda os 6 braços, emite CSV
api/cmd/dense-determinism/main.go       estabilidade do ranking por fator
scripts/embed-corpus.py                 gera os embeddings (uma vez, versionado)
scripts/dense-floor-diagnostic.py       análise de separabilidade do piso
scripts/analyze.py                      Wilcoxon, Bonferroni, Cliff's δ, bootstrap → LaTeX
scripts/test_analyze.py                 19 testes do script de análise
docs/article/results/results.csv        saída bruta (uma linha por consulta × braço)
docs/article/tables/*.tex               tabelas prontas para \input{} no Overleaf
```
