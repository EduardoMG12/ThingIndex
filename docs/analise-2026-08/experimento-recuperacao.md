# Experimento comparativo de recuperação — metodologia e resultados

> **Estado: implementado e executável.** Tudo aqui é reproduzível com dois
> comandos. Nenhum número deste documento foi transcrito à mão.
>
> ```bash
> cd api && go run ./cmd/retrieval-eval -out ../docs/article/results/results.csv -summary
> python3 scripts/analyze.py docs/article/results/results.csv --outdir docs/article/tables
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
| **D — RRF** | Fusão de A e B por Reciprocal Rank Fusion, `k=60` (Cormack et al., 2009) | `internal/retrieval/metrics.go` |

O braço A é **reprodução fiel** do scorer de produção, defeitos inclusive. Isso
é o ponto: o experimento mede o sistema como ele é, não uma versão idealizada.

> **O braço C (denso, pgvector + embeddings) não está implementado, e a ausência
> é declarada em vez de escondida.** Ele exige um modelo de embedding rodando, o
> que tira a execução do regime hermético e reproduzível em CI. A estrutura já o
> acomoda — basta implementar a interface `arm` em `cmd/retrieval-eval`.
> Prometer um braço que não roda seria repetir exatamente o erro do Apêndice A,
> que anunciava arquivos DBML inexistentes.

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
| A — relacional | 0,750 | 0,830 | 0,869 | **1,000** |
| B — BM25 | 0,812 | 0,862 | 0,894 | **1,000** |
| D — RRF | 0,812 | 0,870 | 0,900 | **1,000** |

### P@1 por variante de consulta

| Braço | Literal | Paráfrase | Ruidosa | Queda literal→paráfrase |
|---|---|---|---|---|
| A — relacional | 1,000 | 0,571 | 0,750 | **−0,429** |
| B — BM25 | 1,000 | 0,714 | 0,750 | **−0,286** |
| D — RRF | 1,000 | 0,571 | 1,000 | −0,429 |

---

## 4. Leitura honesta dos resultados

**Achado 1 — a queda literal→paráfrase é o resultado central.** Todos os braços
acertam 100% quando a consulta copia termos do alvo, e despencam para 57–71%
quando a mesma necessidade é expressa com outras palavras. Isso quantifica
exatamente o que o benchmark antigo escondia: os 100/100/100 mediam casamento
lexical, não recuperação. **A queda é maior no braço relacional (−0,429) do que
no BM25 (−0,286)**, o que é coerente com a ausência de IDF e de normalização por
comprimento no scorer atual.

**Achado 2 — nenhum braço reconhece a ausência de resposta.** Taxa de falso
positivo **1,000 nos três**: em toda consulta cujo gabarito é "nada é
relevante", todos devolveram algo. Esta é uma limitação real do sistema, medida
pela primeira vez, e não aparecia em nenhuma métrica de ranking.

**Achado 3 — as diferenças entre braços NÃO são estatisticamente
significativas.** Todos os *p* corrigidos dão 1,000 e todos os Cliff's δ são
desprezíveis. **Isso é uma limitação de poder amostral, não evidência de
empate.** Com 16 consultas positivas, poucos pares diferem e o Wilcoxon não tem
como detectar efeito. O script devolve `p = 1.0` explicitamente quando *n* < 10
em vez de fabricar um número que parece preciso.

> **Consequência direta para o TCC II:** o corpus precisa crescer para ~150
> consultas antes que qualquer afirmação comparativa seja defensável. A
> infraestrutura está pronta e testada; o que falta é anotação, que é trabalho
> humano e não comprime no fim do semestre.

---

## 5. O que fica pendente

| Item | Por quê |
|---|---|
| Corpus para ~150 consultas, ≥30 documentos | Sem isso não há poder estatístico (§4, achado 3) |
| Dupla anotação em ≥20% com Cohen's κ reportado | Sem κ, "gabarito" é opinião do autor |
| Braço C (pgvector + `multilingual-e5-base`) | Sem ele não há comparação com RAG denso — o alvo declarado do artigo |
| Documentos escaneados (curva CER × dpi) | O caminho de OCR nunca foi medido |
| Métricas de auditabilidade (TPC, TCV, determinismo) | São a contribuição própria do trabalho, e a vitória mais barata disponível |

---

## 6. Arquivos

```
api/internal/retrieval/metrics.go      métricas de RI + RRF
api/internal/retrieval/lexical.go      braços A (substring) e B (BM25)
api/internal/evalcorpus/corpus.go      carga, validação e checagem de circularidade
api/internal/evalcorpus/testdata/      corpus C1
api/cmd/retrieval-eval/main.go         harness: roda os braços, emite CSV
scripts/analyze.py                     Wilcoxon, Bonferroni, Cliff's δ, bootstrap → LaTeX
scripts/test_analyze.py                19 testes do script de análise
docs/article/results/results.csv       saída bruta (uma linha por consulta × braço)
docs/article/tables/*.tex              tabelas prontas para \input{} no Overleaf
```
