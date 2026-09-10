# Conexões por token exato, sem números, com teto de grau e aresta tipada "mesma norma"

**Status:** em execução (2026-09-09), branch `feat/conexoes-por-token` a partir de `feat/tabelas-de-pdf` (PR #71, sem merge)
**Data:** 2026-09-09
**Origem:** veredito do conselho (item 5) sobre o que o autor viu na tela: arestas justificadas por "nº, resolução", nomes próprios e pedaços de palavra. Três causas no código: `rareTerms` e `matchedTerms` casam por substring (`strings.Contains`: "lei" casa "leitura", "estágio" casa "estagiário"); tokens numéricos contam como termo (anos, números de resolução); e um nó pode ter 160 arestas sem nada o limitar.

## Goals

(1) Um termo só sustenta conexão se aparece como **token inteiro** no destino, contado por token também na raridade. (2) Token só de dígitos (ou dígitos com pontuação: "9.394", "82/2022") não conta como termo raro. (3) Um nó tem no máximo `NodeConnectionMaxDegree = 20` arestas, as de maior score. (4) Aresta tipada `mesma-norma`: quando a origem cita uma norma ("Lei nº 9.394/1996", "Resolução CONSUP 82/2022", "Decreto 7.824/2012"), o destino que cita a mesma norma ganha uma conexão com `kind = "mesma-norma"` e `matched_terms = [citação normalizada]`, mesmo sem termo raro em comum. Tudo continua derivado, determinístico e sem persistência (constituição II).

Pronto quando: unitários provam substring não casa ("lei" × "leitura"), número não sustenta, teto de grau corta, e citação de norma casa entre documentos; na amostra de 60 seções da coleção de SI, registrada antes e depois (`scripts`-free: `conn-sample.py` no relatório), a contagem de arestas e os termos que as justificam ficam registrados em Discoveries.

## Requirements

- [ ] `tokenSet(text)`: tokens de `tokenizeRetrievalText` num `map[string]struct{}`; `rareTerms` conta df por pertinência ao conjunto; `matchedTerms` e o score de conexão usam o conjunto (score = número de termos raros presentes; `minScore` continua 2).
- [ ] `ehPalavraVazia` recusa token numérico: `^[\d.,/-]+$`.
- [ ] `NodeConnectionMaxDegree = 20`: após ordenar, corta a lista.
- [ ] `normCitations(text) []string`: regex sobre o texto cru da origem e do destino, `(lei|resolução|resolucao|decreto|portaria|parecer|instrução normativa)\s*(n[º°o.]?\s*)?([\d.]+)(/(\d{2,4}))?`, normalizada para `tipo:numero/ano` (número sem pontos; ano só se houver). Origem com citações gera conexão `mesma-norma` para candidato de outro documento com citação igual; se o candidato já tem conexão por termo raro, a conexão recebe `Kind = "termo-raro+mesma-norma"`? Não: uma aresta por par; `Kind` = `mesma-norma` quando a citação casa, senão `termo-raro`; `MatchedTerms` junta os dois conjuntos.
- [ ] `domain.NodeConnection.Kind`, `dto.NodeConnectionResponse.Kind` (`kind`), contrato do frontend com `kind` opcional; a legenda da aresta mostra "mesma norma" quando for o caso.
- [ ] Testes novos em `node_connections_test.go`; amostra de 60 antes/depois em Discoveries.

## Non-Goals

- Excluir nomes próprios por caixa: sem o texto cru posicional não dá para separar "Bruno" de "Estágio" no início da frase; fica registrado.
- Persistir arestas; mudar o desenho do canvas além do rótulo do tipo.

## Constraints

- `tokenizeRetrievalText` é o tokenizador de produção (com expansão numérica); `FoldDiacritics` já dobra acento no retrieval, mas o tokenizador das conexões hoje só baixa caixa: manter, e dobrar acento nas duas pontas do casamento de token.
- `retrievalScore` (braço A) continua intacto: as conexões deixam de usá-lo.
- Frontend: `src/lib/api/contracts/node-connections.ts` valida o contrato; `src/lib/workspace/connections.ts` monta o rótulo.

## Ambiguity Report (grill-me, modo spec, 2026-09-09)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (amostra e registro, nao criterio)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (nomes proprios adiados com razao)
  Assumptions:  0.0   ✓ clear (verificado: Contains em rareTerms/matchedTerms/retrievalScore; DTO e contrato do frontend)
  ──────────────────────────────
  Aggregate:    0.10  ✓ below threshold (0.2 spec)
```

## Discoveries

- **Amostra de 60 seções da coleção de SI (2026-09-09), mesmo script antes e depois.** Antes: 56 seções com conexão, 3.444 arestas, grau máximo 204; os termos que mais justificavam arestas eram "objetivos", "conceitos", "língua", "paulo", "marini", "andreia", "1º". Depois (token inteiro, sem número, sem nome próprio em caixa de título, teto de 2% de raridade e 20 arestas por nó): 37 seções com conexão, 443 arestas, grau máximo 20; termos mais frequentes "alunos", "participação", "colegiados", "tanenbaum", "elmasri", "navathe" (bibliografias em comum), "matriculados", "indígenas". Sobra ruído de marcador de lista ("e/ou") e de abreviação ("rev").
- O teto de raridade caiu de 5% para 2% junto com a troca para token inteiro: por substring o df era inflado ("lei" contava "leitura") e 5% compensava; por token, 5% deixava "objetivos" passar.
- Nome próprio: excluído quando no texto da origem a palavra só aparece com inicial maiúscula e resto minúsculo (fora da primeira posição). Palavra em CAIXA ALTA (título) continua valendo. Custo conhecido: palavra que só ocorre no início de frase também sai.
- A reindexação sem crédito de LLM (fallback de resumos) foi o que fez as conexões explodirem entre a rodada 3 do PRD anterior e esta amostra "antes"; a comparação antes/depois aqui é sobre o mesmo índice.
- **Depois da reindexação com enriquecimento (2026-09-09), mesma amostra de 60 seções:** 355 arestas. Com numerais romanos ("vi") e aparato bibliográfico ("isbn", "ed", "editora") fora, mais "ações/atender/conjunto/geral" e "desta/deste" como vazias: **307 arestas, 31 seções com conexão, grau máximo 20**; os termos que mais justificam passam a ser "sustentabilidade", "soluções", "inteligência artificial", "deitel" (bibliografia em comum), "colegiados", "autonomia".
