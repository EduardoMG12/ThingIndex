# Proveniência: página no chunk, página real na tabela e TPC por escopo

**Status:** proposto (2026-09-19), branch `feat/proveniencia-completa-e-multidominio`
**Data:** 2026-09-19
**Origem:** o conselho consultado em 2026-09-18 apontou a TPC de 0,329 como a métrica mais barata de corrigir e a mais central para a tese ("auditabilidade"). A apuração no código mostrou que parte do número está errada por defeito do avaliador, não do sistema.
**Autor ausente:** o PRD foi escrito sem entrevista. Cada decisão que normalmente seria pergunta está marcada como **Assunção** para revisão.

## Contexto apurado (não é suposição)

O artigo (`docs/article/article-overleaf.md:660`) reporta TPC global de 0,329 com leitura bimodal: 1,000 em parágrafo, página e nó de sumário; 0,000 em chunk, tabela e metadado. Atribui a causa a "as entidades `TableSection`, `MetadataPair` e `DocumentChunk` não possuem coluna de página".

Três achados contradizem parcialmente esse texto:

1. **`models.TableSection` já tem `PageNumber *int`** com índice (`api/internal/models/document.go:91`), acompanhado de comentário explicando que tabela que atravessa páginas vira uma `TableSection` por página. O avaliador, porém, ainda passa `PageNumber: nil` fixo para tabelas (`api/cmd/provenance-eval/main.go:116-123`) com um comentário afirmando que o esquema não tem a coluna. O comentário do avaliador está obsoleto. **Ressalva apurada no portão de ambiguidade: isso não altera o número de C1**, porque o corpus C1 é texto puro e não produz tabela nenhuma. É higiene de medição, para que um corpus futuro com tabela não seja medido errado.

2. **`models.DocumentChunk` não tem página, mas a página existe na origem.** Os chunks nascem em `api/internal/indexing/chunks.go`: o caminho principal fatia nós de sumário (`DocumentOutlineNode`, que tem `StartPage` e `EndPage`) e o fallback fatia páginas (`DocumentPage.PageNumber`). O chunk também guarda `SourceParagraphStart/End`, e `ParagraphNode` tem `PageNumber`. A página não precisa ser inferida: ela está disponível no momento da construção.

3. **`MetadataPair` não é evidência citável.** Os pares criados em `api/internal/indexing/outline.go:85-105` são estatísticas do documento inteiro (`paragraph_count`, `table_count`, `tree_builder`, `tree_coverage_ratio`). No chat eles alimentam apenas `buildDocumentSummaryMap` (`api/internal/usecase/chat_usecase.go:853,956`), nunca `buildChatSources`. Já os chunks entram em `buildCollectionRetrieval` (`chat_usecase.go:171`) e viram fonte citada. Ou seja: chunk precisa de página porque é citado; par de metadado não é citado e não existe em página nenhuma.

O princípio I da constituição exige rastrear "documento, página, posição sequencial e tipo de estrutura". A tensão com o item 3 é real e a resolução proposta abaixo é o ponto que mais precisa de revisão humana.

## Goals

(1) O avaliador para de passar `nil` fixo para tabela e passa a usar a página que o modelo já carrega, de modo que uma medição futura sobre corpus com tabela não reporte 0,000 por defeito de instrumento. (2) `DocumentChunk` passa a carregar a página de origem, preenchida a partir da estrutura que o gerou, nunca inferida por heurística. (3) A TPC passa a ser reportada por **escopo** (página ou documento), de modo que uma estatística do documento inteiro não seja contada como falha de proveniência de página, e nenhuma página seja fabricada para satisfazer a métrica. (4) A subseção de auditabilidade do artigo passa a reportar os números novos, com a causa da mudança declarada.

Pronto quando: `go test ./...` passa com os valores esperados atualizados; `cmd/provenance-eval` roda sobre o corpus C1 e imprime TPC por tipo **e** por escopo; o novo número aparece no artigo copiado com uma frase explicando por que mudou; nenhum item recebe página que não esteja na estrutura de origem.

## Requirements

- [ ] `models.DocumentChunk` ganha `PageNumber *int` com `gorm:"index"`, nulável, `json:"page_number,omitempty"`. O `AutoMigrate` de `appdb.ManagedModels()` cria a coluna; não há arquivo de migração (ver Descoberta 1).
- [ ] `chunkBuilder.buildChunksForNode` e os dois retornos de `splitOversizedChunk`: `PageNumber` recebe a página do primeiro parágrafo não vazio do intervalo do nó; se nula, `node.StartPage`; se também nula, fica nulo, nunca zero (ver Descoberta 3).
- [ ] `chunkBuilder.buildFallbackFromPages`: `PageNumber` recebe a página do `DocumentPage` de origem.
- [ ] `provenance.Item` ganha `Scope string` com dois valores: `"page"` (a evidência existe em uma página do documento) e `"document"` (o item descreve o documento inteiro).
- [ ] `provenance.Evaluate`: item de escopo `document` tem proveniência completa com `DocumentID != 0`, `SequenceOrder >= 0` e conteúdo não vazio, sem exigir página. Item de escopo `page` mantém o critério atual, incluindo página.
- [ ] `Report` ganha `ByScope map[string]KindReport` e `RateForScope(scope string) float64`, espelhando `ByKind`/`RateFor`.
- [ ] `cmd/provenance-eval`: tabela passa a usar `table.PageNumber`; chunk passa a usar `chunk.PageNumber`; par de metadado entra com escopo `document`; parágrafo, página, tabela, chunk e nó de sumário entram com escopo `page`. O comentário obsoleto sobre "o esquema não tem a coluna" é corrigido, não apagado: fica registrado que a coluna passou a existir e quando.
- [ ] A saída de `cmd/provenance-eval` imprime TPC global, por tipo e por escopo, nessa ordem.
- [ ] `api/cmd/provenance-eval/main_test.go`: as constantes esperadas (hoje `tpcGlobalC1 = 0.329`) passam para os valores **derivados da definição**, não copiados da saída: TPC global de C1 = 1,000 e cada `RateFor` = 1,000, pela derivação registrada no portão de ambiguidade. O teste continua travando valor exato, e o teste que hoje exige a existência de um tipo incompleto (`TestTPCReportaPorTipoEstruturalENaoSoAMedia`) passa a exigir o contrário, com o motivo escrito nele.
- [ ] `api/internal/provenance/provenance_test.go`: casos novos para escopo `document` (completo sem página) e para escopo `page` (incompleto sem página). O caso existente que espera `RateFor("table") == 0.0` é atualizado junto com a causa registrada no teste.
- [ ] A cópia de trabalho do artigo (`docs/article/article-overleaf-v2.md`) tem a subseção de auditabilidade reescrita com os números novos, a distinção de escopo e a frase explicando que a medição anterior subestimava as tabelas.

## Non-Goals

- Não adicionar `PageNumber` a `MetadataPair`. Ver Assunção 1.
- Não mexer no cálculo da TCV (taxa de citação verificável), que hoje está em 1,000 e não tem defeito conhecido.
- Não reindexar as coleções de demonstração como parte desta entrega. A coluna nova fica nula nos documentos já indexados até que sejam reprocessados; isso é aceitável porque a métrica do artigo é medida por `cmd/provenance-eval`, que indexa o corpus C1 em memória a cada execução e não lê o banco.
- Não alterar a recuperação nem o ranking. Nenhuma consulta passa a usar a coluna nova nesta entrega.
- Não tocar em `cmd/retrieval-eval`, `cmd/threshold-eval` nem `cmd/dense-determinism`.

## Constraints

- **Constituição, princípio I:** proveniência é constraint de esquema, e é proibido fabricar origem. Preencher página por heurística (por exemplo "chunk herda a página do primeiro parágrafo parecido") está fora.
- **Constituição, princípio II:** a mesma entrada produz a mesma saída. A página do chunk vem da estrutura de origem, então é determinística por construção.
- `cmd/provenance-eval` usa o indexador real (`indexing.NewDeterministicIndexer`) sobre texto puro, não uma simulação (`main.go:62-88`). O corpus C1 é texto, então o caminho exercitado é o do `.txt`, no qual `DocumentPage` existe e `OutlineNode.StartPage` pode ser nulo em documentos sem sumário. O comportamento com `StartPage` nulo precisa de teste explícito.
- O número muda um resultado já escrito no artigo. A mudança só é honesta se vier acompanhada da explicação da causa, senão parece maquiagem de métrica.
- O artigo original não pode ser editado: toda escrita vai para a cópia `article-overleaf-v2.md`.

## Approach

Três mudanças pequenas e independentes, nesta ordem:

1. **Avaliador primeiro, sem tocar no esquema.** Trocar o `nil` fixo da tabela pelo `table.PageNumber`. **Correção importante em relação à primeira versão deste PRD:** isso **não** muda a TPC de C1. O teste travado (`cmd/provenance-eval/main_test.go:25-28`) declara que os tipos medidos em C1 são cinco, `paragraph`, `page`, `outline` (completos) e `chunk`, `metadata` (incompletos). Tabela não está na lista porque o corpus C1 é texto puro e não produz `TableSection`. Logo a correção do `nil` é higiene de medição, não mudança de número: ela impede que uma medição futura sobre corpus com tabela reporte 0,000 por defeito do avaliador, e corrige o comentário que afirma que o esquema não tem a coluna. O que de fato muda o número é a página no chunk e o escopo do metadado.
2. **Escopo.** Introduzir `Scope` em `provenance.Item`, o critério dependente de escopo em `Evaluate`, e `ByScope` no `Report`. Atualizar o avaliador para classificar cada tipo.
3. **Página no chunk.** Modelo, migração, preenchimento nos dois caminhos de `chunks.go`, teste unitário do builder.

Depois das três, rodar `cmd/provenance-eval` uma vez e registrar os números finais no artigo copiado.

O padrão de teste a seguir é o que o projeto já usa nessas métricas: valor esperado calculado à mão a partir da definição, nunca copiado da saída da implementação (`api/internal/provenance/provenance.go:1-20` explica a regra).

## Assunções (revisar)

1. **`MetadataPair` não recebe coluna de página, e sim escopo de documento.** Razão: os pares que o indexador cria são estatísticas do documento (`paragraph_count`, `tree_builder`), que não existem em página nenhuma, e no chat eles não viram fonte citada. Contra-argumento que o revisor pode levantar: a rota `POST /documents/:id/metadata-pairs` (`api/internal/handlers/document.go:864`) permite que o usuário grave pares arbitrários, e um par gravado por usuário *poderia* ser específico de uma página. Se isso passar a importar, a saída natural é `Scope` no próprio par, não uma coluna de página obrigatória.
2. **Chunk fatiado de nó de sumário recebe `StartPage`, não um intervalo.** Um chunk pode atravessar páginas, e `StartPage` é a página onde ele começa. A alternativa seria guardar `StartPage` e `EndPage` no chunk. Escolhi a coluna única porque a citação do chat aponta uma página para o usuário clicar, e porque `TableSection` já resolveu o caso equivalente com uma coluna só.
3. **O teste continua travando um valor exato.** Alternativa rejeitada: afrouxar para faixa. O propósito declarado do teste é falhar quando o número muda sem ninguém perceber.
4. **Documentos já indexados ficam com a coluna nula.** Não haverá backfill nesta entrega. Se o revisor quiser o backfill, ele é um `UPDATE ... FROM` juntando `document_chunks` com `paragraph_nodes` por `source_paragraph_start`, e vira issue separada.

## Open Questions

- **(decisão do usuário, única pendência real)** A mudança de critério (escopo) é defensável diante da banca ou é vista como afrouxar a métrica para melhorar o número? Minha leitura: é defensável **se** o artigo publicar os três números lado a lado (0,329 do critério antigo, o valor com a correção do avaliador, e o valor por escopo) e explicar cada um. Publicar só o número final seria o caso ruim. Implementei publicando os três.

## Discoveries

Apuradas antes da execução, resolvendo duas das três perguntas abertas originais:

1. **Não existe migração explícita.** O esquema é mantido por `AutoMigrate` sobre `appdb.ManagedModels()` (`api/db/connect.go:54`, `api/db/models.go`), e `DocumentChunk` já está na lista (`db/models.go:37`). Adicionar o campo ao modelo basta; não há arquivo de migração a escrever. Isso remove um requisito do PRD original.

2. **No corpus C1 tudo cai na página 1.** Os documentos de C1 são texto puro sem `\f` (verificado em `internal/evalcorpus/testdata/corpus-c1.json`), e `buildDocumentPages` (`internal/indexing/outline.go:139-169`) atribui `pageNumber = 1` quando o bloco não traz página. **Consequência séria para a leitura do resultado:** a TPC de chunk medida em C1 sobe porque o mecanismo passa a carregar a página, mas C1 não prova que a página atribuída é a *certa* em documento de várias páginas, porque em C1 só existe uma. O artigo precisa declarar isso, e a prova de atribuição correta tem que vir de teste unitário com parágrafos espalhados por páginas diferentes, não da TPC de C1. Isso vira requisito novo:

- [ ] Teste unitário em `internal/indexing/chunks_test.go` com parágrafos nas páginas 1, 2 e 3 provando que cada chunk recebe a página do seu primeiro parágrafo de conteúdo, e que chunk de nó sem parágrafo com página fica nulo em vez de 1.

3. **A página do chunk deve vir do parágrafo, não do nó.** `buildChunksForNode` já resolve o intervalo de parágrafos (`outlineParagraphRange`) e tem acesso a `[]models.ParagraphNode`, que carrega `PageNumber` real. `node.StartPage` é derivado dos mesmos parágrafos, então usar o parágrafo é a origem mais próxima e evita um passo de derivação. Ordem de precedência adotada: página do primeiro parágrafo não vazio do intervalo; se nula, `node.StartPage`; se também nula, nulo.

4. **São quatro locais de criação de chunk, não dois.** `buildChunksForNode` (tipo `outline`), dois retornos dentro de `splitOversizedChunk` (tipo `outline-split`) e `buildFallbackFromPages` (tipo `page`). O split não recebe parágrafos hoje, então a página precisa ser passada como parâmetro para não recalcular.

## Portão de ambiguidade (grill-me, modo spec, limiar 0,2)

Interrogatório feito sem o usuário, respondendo cada dimensão com leitura de código. Três achados mudaram o PRD e estão incorporados acima.

### O que o interrogatório corrigiu

1. **A correção da tabela não muda número nenhum em C1.** Eu tinha escrito que ela "isola quanto do 0,329 era defeito de medição". Errado: `cmd/provenance-eval/main_test.go:25-28` declara os cinco tipos medidos em C1 e tabela não está entre eles, porque texto puro não gera `TableSection`. A afirmação foi reescrita.

2. **O artigo erra na causa, não no número.** O texto em `article-overleaf.md:660` reporta 0,000 para "chunks e metadados" (correto para C1) mas atribui a causa a "`TableSection`, `MetadataPair` e `DocumentChunk` não possuem coluna de página". `TableSection` tem a coluna desde que as tabelas de PDF viraram fonte. A frase da causa é o que precisa mudar no artigo, junto com os números novos.

3. **O valor esperado do teste não pode ser copiado da saída.** A regra do projeto (`internal/provenance/provenance.go:1-20`) é calcular o esperado à mão a partir da definição. Derivação sem rodar nada: depois da mudança, todo item de escopo `page` em C1 tem página (texto puro cai todo na página 1, `internal/indexing/outline.go:162-165`) e todo item de escopo `document` é completo por definição. Logo **TPC global esperada em C1 = 1,000**, e cada `RateFor` esperado = 1,000. O teste passa a travar esses valores derivados, não um número observado.

### Alternativas examinadas para o `MetadataPair` (a decisão mais frágil)

| Alternativa | Por que foi descartada |
|---|---|
| Adicionar `PageNumber` ao par e preencher com a página onde a estatística "aparece" | Fabricaria proveniência. `paragraph_count` não está em página nenhuma. Viola o princípio I pelo avesso: cumpre a letra inventando o dado. |
| Tirar o par do denominador da TPC | Esconde o item em vez de classificá-lo. Um leitor não saberia que ele existe, e a métrica passaria a medir só o que convém. |
| **Escopo explícito (escolhida)** | Mantém o item visível, declara por que o critério dele é outro, e deixa o leitor conferir os dois recortes. Custa uma coluna a mais no relatório. |
| `Scope` no próprio `MetadataPair`, e não no item de avaliação | Seria necessário se os pares gravados pelo usuário via `POST /documents/:id/metadata-pairs` fossem específicos de página. Hoje nenhum é. Fica registrado como o caminho natural se isso mudar. |

### Relatório

```
Ambiguity Report:
  Goals:        0.25  ✓ claro (alvo numérico agora é derivável: TPC C1 = 1,000)
  Acceptance:   0.25  ✓ claro (valores esperados derivados, não observados)
  Boundaries:   0.0   ✓ claro (Non-Goals enumerados, incluindo o que não reindexar)
  Alternatives: 0.25  ✓ claro (quatro caminhos examinados para o metadado)
  Assumptions:  0.25  ✓ claro (quatro assunções nomeadas, três verificadas em código)
  ──────────────────────────────
  Aggregate:    0.20  ✓ no limiar (0,2 spec)
```

### Pendência real para revisão humana

Uma só, e é de mérito, não de forma: **TPC de 1,000 em C1 é um resultado fraco e precisa ser apresentado como tal.** C1 é texto puro de uma página, então a métrica não consegue distinguir "a página certa" de "a constante 1". A prova de atribuição correta passa a ser o teste unitário com parágrafos em páginas diferentes, e o artigo precisa dizer isso na mesma frase em que publica o 1,000. Se o revisor achar que publicar 1,000 parece maquiagem, a alternativa é medir a TPC sobre uma coleção real de PDF já indexada no banco, o que exige `cmd/provenance-eval` ler do banco em vez de indexar em memória. Isso é issue separada, não cabe nesta.
