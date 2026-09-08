# Spec 001 — Recuperação em nível de nó e Workspace de proveniência

**Status:** proposta
**Data:** 2026-09-04
**Branch alvo:** `feat/recuperacao-nivel-no`
**Constituição aplicável:** `.specify/memory/constitution.md` (princípios I, II, III, VI)

---

## 1. Problema

O ThingIndex afirma, no artigo e na constituição, que sua contribuição é
**proveniência fina**: cada trecho recuperado sabe de que documento, página e
posição veio. O experimento comparativo, porém, **ranqueia documentos inteiros**
(`cmd/retrieval-eval/main.go:199-208`, o scorer recebe `docs` e devolve um
ranking de documentos).

Isso produz três defeitos encadeados, todos verificados:

1. **Descasamento entre tese e medição.** A tese fala de parágrafo; o experimento
   mede documento. É a objeção mais barata que uma banca pode fazer, e nenhuma
   reescrita de texto a resolve.
2. **Métricas saturadas por construção.** 14 das 16 consultas positivas têm
   exatamente 1 documento relevante entre 5, então `R@5 = 1,000` é aritmético, não
   resultado; e `P@3` tem teto de 0,375 sem que a tabela declare esse teto.
3. **Capacidade instalada ociosa.** `domain.TreeRetrievalResult` já define
   `NodeHits []TreeNodeHit` e `PageHits []TreePageHit`, exposto em
   `GET /documents/:id/index`. O sistema **já recupera em nível de nó em
   produção** — e o experimento ignora isso.

Existe ainda uma limitação medida e não endereçada: **taxa de falso positivo
1,000 nos seis braços**. Nenhum sabe responder "não encontrei".

## 2. Objetivo

Mover a unidade de recuperação do experimento de **documento** para **nó
estrutural** (`DocumentOutlineNode`), e expor o resultado disso numa interface
que torna a proveniência visível e navegável.

Não é uma feature de produto com pesquisa por cima. É o contrário: é a correção
do experimento, e a interface é a sua visualização.

## 3. Escopo

### P1 — Recuperação em nível de nó no experimento (independentemente entregável)

O braço relacional passa a ranquear nós, não documentos. Corpus e gabarito passam
a referenciar `node_id` além de `document_id`.

**Por que P1:** sozinho, já conserta o descasamento tese×experimento e produz uma
seção de Resultados nova. Se o semestre acabar aqui, o TCC melhorou.

**Como testar isolado:** `go run ./cmd/retrieval-eval` emite CSV com uma linha por
(consulta, braço, nó); `scripts/analyze.py` gera as tabelas; os testes de
anti-circularidade continuam passando.

**Cenários:**
- **Dado** uma consulta com alvo conhecido em nível de nó, **quando** o braço
  relacional roda, **então** o ranking devolve `node_id`s e a métrica é calculada
  sobre eles.
- **Dado** o gabarito antigo (por documento), **quando** o harness roda, **então**
  ele **aborta** com mensagem explícita em vez de emitir número comparando
  unidades diferentes.
- **Dado** um teto de métrica atingível (`P@3` limitado pelo número de nós
  relevantes), **quando** a tabela é gerada, **então** o teto é publicado ao lado
  do valor.

### P2 — Métricas de auditabilidade no artigo (independentemente entregável)

`internal/provenance` já implementa TPC (taxa de proveniência completa) e TCV
(taxa de citação verificável), com testes. Nenhuma das duas aparece no artigo,
embora a Metodologia prometa avaliar auditabilidade.

**Como testar isolado:** um comando emite a tabela `.tex`; o CI falha se o número
no texto divergir do medido (constituição, princípio III).

**Cenários:**
- **Dado** o corpus indexado, **quando** o relatório de proveniência roda,
  **então** a TPC é reportada **por tipo estrutural**, não só na média global.
- **Dado** um tipo estrutural sem `page_number` (ex.: blocos de OCR sem `Box`),
  **quando** o relatório roda, **então** ele aparece com TPC baixa em vez de ser
  diluído na média.

### P3 — Limiar de conexão medido (independentemente entregável)

Hoje o falso positivo é 1,000: todo trecho conecta com tudo. Medir a
separabilidade dos scores entre pares relevantes e irrelevantes responde "a
partir de que score uma conexão é real?".

**Como testar isolado:** script de diagnóstico emite as distribuições e o limiar
ótimo (ou demonstra que não existe limiar separador — que também é resultado).

**Cenários:**
- **Dado** o conjunto de pares (trecho, nó) com relevância conhecida, **quando** o
  diagnóstico roda, **então** ele reporta as distribuições de score e o ponto de
  corte que maximiza a separação.
- **Dado** que as distribuições se sobrepõem completamente, **quando** o
  diagnóstico roda, **então** ele reporta isso como achado e **não** inventa um
  limiar arbitrário.

### P4 — Workspace de proveniência (UI)

Aba nova: painel navegável com os documentos da coleção como cartões. Ao focar um
documento, sidebar direita com o outline hierárquico e os resumos que **já
existem** (`DocumentOutlineNode.Summary`, `DocumentPage.Summary`). Clicar numa
seção salta para a página.

Sobre um nó focado, o botão "conexões" usa o **texto do nó como consulta** contra
o restante da coleção e desenha as arestas resultantes, cada uma com score e
justificativa ("estes termos casaram, neste parágrafo").

**Por que P4 e não P1:** é a visualização de P1 e P3. Sem eles, é tela bonita sem
resultado por trás.

**Cenários:**
- **Dado** um documento indexado, **quando** o usuário o foca, **então** a sidebar
  mostra o outline com resumos, sem nenhuma chamada de LLM.
- **Dado** um nó focado, **quando** o usuário pede conexões, **então** as arestas
  são **calculadas na hora** e cada uma exibe score e trecho que a justificou.
- **Dado** que nenhuma conexão passa do limiar de P3, **quando** o usuário pede
  conexões, **então** a UI diz "nenhuma conexão acima do limiar" — e **não**
  desenha a melhor mesmo assim.

## 4. Fora de escopo, com motivo

- **Tabela de arestas persistida entre documentos.** A constituição, princípio II,
  permite grafo apenas como trilha comparativa, "nunca como dependência do caminho
  principal". Aresta armazenada e percorrida para recuperar É camada primária de
  grafo — contradiz a pergunta de pesquisa registrada no TCC I. As arestas deste
  spec são **derivadas**: recalculadas por consulta, nunca persistidas.
- **Extrator de referências entre documentos** ("este documento revoga aquele").
  É inferência jurídica (vigência, hierarquia, competência), não recuperação.
- **Anotação de gabarito de conexões trecho→trecho.** O corpus C1 é sintético e
  autoral; anotar as conexões seria circular. As conexões entram como análise
  qualitativa declarada, não como precisão medida.
- **Renderização de PDF no cartão via pdf.js.** `DocumentPage.Content` já está no
  banco. Dependência nova por ganho estético.
- **Braço HNSW.** Continua em Trabalhos Futuros.

## 5. Requisitos não funcionais

1. **Degradação:** se a recuperação por nó falhar ou o corpus não tiver outline, o
   sistema volta ao comportamento atual (ranking por documento) em vez de quebrar.
2. **Sem LLM no caminho:** a sidebar e as conexões usam apenas dado já persistido e
   SQL. Constituição, princípio II: o sistema funciona por completo sem provedor
   de LLM configurado.
3. **Determinismo:** duas execuções da mesma consulta produzem o mesmo ranking de
   nós, com desempate explícito. Travado por teste.
4. **Isolamento:** toda consulta nova é escopada por `user_id`, com teste que prove
   que o usuário B não alcança nós do usuário A.

## 6. Ponto de integração

O experimento muda em **um lugar**: `api/cmd/retrieval-eval/main.go:199-208`, onde
hoje os braços recebem `docs` e devolvem ranking de documentos. A unidade passa a
ser nó; o restante do harness (métricas, análise estatística, CSV) não muda de
forma, só de conteúdo.

Na aplicação, a capacidade já existe em `domain.TreeRetrievalResult` e é servida
por `GET /documents/:id/index`.

## 7. Decisões tomadas sem o usuário (revisar)

1. **Arestas derivadas, não persistidas.** Rejeitei a tabela de arestas porque a
   constituição a proíbe como camada primária. Alternativa rejeitada: persistir com
   um campo "origem: derivada" — continuaria sendo grafo percorrível.
2. **Sem anotação de conexões agora.** Rejeitei anotar o corpus porque o autor
   escreveu os documentos; seria circularidade, exatamente o defeito que o
   benchmark antigo tinha.
3. **Limiar medido, não escolhido.** Rejeitei fixar um valor plausível (0,5) porque
   o piso de similaridade do braço denso já foi refutado por medição uma vez.
4. **P4 depende de P1 e P3.** Rejeitei fazer a UI primeiro: sem resultado por trás,
   a banca pergunta qual é a contribuição científica e não há resposta.
5. **Páginas extraídas em vez de pdf.js.** Menos dependência, mesmo valor de uso.

## 8. Riscos

| Risco | Mitigação |
|---|---|
| O corpus C1 tem outline pobre demais para ranquear nós | Medir antes de implementar (spike de 1 dia). Se o outline for raso, P1 vira "ranquear páginas", que ainda é granularidade fina |
| Prazo: ~8 semanas úteis com o artigo por escrever | P1..P4 são independentes. Ordem de corte: P4 primeiro, depois P3 |
| A mudança de unidade invalida as tabelas já geradas | As tabelas são geradas por script a partir do CSV; regenerar é um comando. Nenhum número é transcrito à mão |
| Orientador não viu Resultados/Discussão ainda | Enviar antes de começar a implementar |
