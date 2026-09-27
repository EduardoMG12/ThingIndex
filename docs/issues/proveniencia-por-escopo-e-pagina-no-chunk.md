# Issues: proveniência por escopo e página no chunk

Quebra do PRD `docs/prds/proveniencia-por-escopo-e-pagina-no-chunk.md`.
Branch: `feat/proveniencia-completa-e-multidominio`.

Ordem de execução: **#1 e #2 em paralelo** (não se tocam), depois **#3**, depois **#4**, depois **#5**.

---

## Issue #1 — Escopo de proveniência no pacote `provenance`

**Context**
A TPC trata todo item como se precisasse de página. Estatísticas do documento inteiro (`paragraph_count`, `tree_builder`) não existem em página nenhuma, então hoje entram no denominador como falha permanente. O objetivo é classificar o item por escopo em vez de fabricar uma página para ele.

**What to do**
1. Em `provenance.Item`, adicionar `Scope string` com dois valores válidos: `"page"` e `"document"`. Documentar no comentário do campo por que ele existe (item de escopo de documento não tem página por natureza, e inventar uma seria fabricar proveniência).
2. Definir a constante padrão: item sem `Scope` preenchido é tratado como `"page"`. Isso mantém todo chamador existente com o comportamento atual e evita que um esquecimento vire silenciosamente um item "completo".
3. Em `Evaluate`, aplicar o critério por escopo: `document` exige `DocumentID != 0`, `SequenceOrder >= 0` e conteúdo não vazio; `page` exige tudo isso **mais** `PageNumber != nil`.
4. `MissingPageNumber` só é incrementado para itens de escopo `page`.
5. Adicionar `ByScope map[string]KindReport` ao `Report`, preenchido junto com `ByKind`, e `RateForScope(scope string) float64` espelhando `RateFor`, inclusive a regra de devolver 0 quando não há itens.

**Where to look**
- `api/internal/provenance/provenance.go` (todo o arquivo, ~200 linhas)
- O cabeçalho do pacote explica a filosofia da métrica: manter o tom e a justificativa ao documentar o campo novo.

**Acceptance criteria**
- [ ] `go test ./internal/provenance/...` passa.
- [ ] Teste novo: item de escopo `document` sem página conta como completo.
- [ ] Teste novo: item de escopo `page` sem página conta como incompleto e incrementa `MissingPageNumber`.
- [ ] Teste novo: item com `Scope` vazio é tratado como `page` (prova que o padrão é o critério estrito).
- [ ] `RateForScope("document")` e `RateForScope("page")` devolvem valores coerentes com `ByScope`, e 0 para escopo inexistente.

**Dependencies** none

**Out of scope**
Não alterar `VerifyCitations` nem a TCV. Não mexer em `cmd/`.

---

## Issue #2 — Página no `DocumentChunk`, vinda da origem real

**Context**
Chunk é evidência citável: ele entra em `buildCollectionRetrieval` e vira fonte na resposta do chat. Sem página, a fonte não é clicável até o trecho, que é o que auditabilidade significa neste trabalho. A página já existe na estrutura que gera o chunk, então não precisa ser inferida.

**What to do**
1. Em `models.DocumentChunk`, adicionar `PageNumber *int` com `gorm:"index"` e `json:"page_number,omitempty"`. O `AutoMigrate` de `appdb.ManagedModels()` cria a coluna, e `DocumentChunk` já está na lista (`api/db/models.go:37`): não escrever arquivo de migração.
2. Em `chunkBuilder.buildChunksForNode`, calcular a página uma vez, na seguinte ordem de precedência, e usá-la nos chunks gerados: página do primeiro parágrafo **não vazio** do intervalo `[start, end]`; se nula, `node.StartPage`; se também nula, `nil`. Nunca zero.
3. Passar essa página como parâmetro para `splitOversizedChunk` e preencher os dois `models.DocumentChunk{}` construídos lá dentro.
4. Em `buildFallbackFromPages`, preencher com a página do `DocumentPage` de origem.

**Where to look**
- `api/internal/models/document.go:112-126` (struct `DocumentChunk`)
- `api/internal/indexing/chunks.go` (quatro locais de construção: `buildChunksForNode`, dois dentro de `splitOversizedChunk`, e `buildFallbackFromPages`)
- `outlineParagraphRange` no mesmo pacote resolve o intervalo de parágrafos do nó.
- Padrão a seguir: `models.TableSection.PageNumber` (`document.go:88-91`) já resolveu o caso equivalente, inclusive o comentário explicando por que é nulável.

**Acceptance criteria**
- [ ] `go build ./...` e `go test ./internal/indexing/...` passam.
- [ ] Teste novo em `chunks_test.go`: parágrafos nas páginas 1, 2 e 3, um nó de sumário por página, e cada chunk recebe a página do seu primeiro parágrafo. **Este é o teste que prova atribuição correta**, porque o corpus C1 é de página única e não consegue provar isso.
- [ ] Teste novo: nó cujo intervalo de parágrafos não tem página e cujo `StartPage` é nulo produz chunk com `PageNumber == nil`, e não `0`.
- [ ] Teste novo: chunk de fallback de página recebe a página daquela página.
- [ ] Nenhuma heurística de similaridade é usada para escolher a página.

**Dependencies** none (pode rodar em paralelo com #1)

**Out of scope**
Não preencher a coluna nos documentos já indexados (sem backfill). Não usar a coluna em nenhuma consulta de recuperação.

---

## Issue #3 — Avaliador usa a página real e classifica escopo

**Context**
O avaliador passa `PageNumber: nil` fixo para tabela, com um comentário afirmando que o esquema não tem a coluna. O esquema tem desde que tabelas de PDF viraram fonte. Com o chunk ganhando página em #2 e o escopo existindo em #1, o avaliador precisa refletir os dois.

**What to do**
1. Em `itemsFromResult`, trocar o `nil` fixo da tabela por `table.PageNumber` e o do chunk por `chunk.PageNumber`.
2. Corrigir os comentários obsoletos **sem apagar a história**: registrar que a coluna passou a existir e que o `nil` fixo era defeito do instrumento, não do sistema.
3. Classificar o escopo de cada tipo: `paragraph`, `page`, `table`, `chunk` e `outline` recebem `Scope: "page"`; `metadata` recebe `Scope: "document"`, com comentário explicando que os pares gerados pelo indexador são estatísticas do documento e não existem em página.
4. Na saída impressa, acrescentar o recorte por escopo depois do recorte por tipo, mantendo global primeiro.

**Where to look**
- `api/cmd/provenance-eval/main.go:92-150` (`itemsFromResult`) e a função de impressão no fim do arquivo (por volta de `main.go:230-250`).

**Acceptance criteria**
- [ ] `go run ./cmd/provenance-eval` (ou o alvo equivalente do projeto) imprime TPC global, por tipo e por escopo, nessa ordem.
- [ ] Nenhum `PageNumber: nil` literal permanece em `itemsFromResult`.
- [ ] Os comentários obsoletos sobre "o esquema não tem a coluna" foram corrigidos.

**Dependencies** #1 e #2

**Out of scope**
Não fazer o avaliador ler do banco. Ele continua indexando o corpus C1 em memória.

---

## Issue #4 — Recalibrar as constantes travadas do teste

**Context**
`main_test.go` trava `tpcGlobalC1 = 0.329` e exige explicitamente que exista ao menos um tipo incompleto, com a mensagem "se o esquema ganhou page_number em tabela/metadado/chunk, o artigo precisa ser atualizado e este teste também". Foi exatamente o que aconteceu. O teste precisa virar o contrário, e o valor esperado precisa vir da definição, não da saída.

**What to do**
1. Derivar o valor esperado sem rodar a ferramenta, e escrever a derivação no comentário do teste: em C1 o texto é puro e `buildDocumentPages` atribui página 1 a todo bloco sem página (`internal/indexing/outline.go:162-165`), logo todo item de escopo `page` tem página; todo item de escopo `document` é completo por definição; portanto **TPC global esperada = 1,000** e cada `RateFor` esperado = 1,000.
2. Atualizar `tpcGlobalC1` para 1.000 e as constantes `tiposCompletos` / `tiposIncompletos`.
3. Inverter `TestTPCReportaPorTipoEstruturalENaoSoAMedia`: ele passa a exigir que **nenhum** tipo esteja incompleto, e a mensagem de falha passa a dizer que uma regressão de proveniência voltou a aparecer.
4. Manter `TestCausaDaIncompletudeEhPaginaAusente` funcionando quando não há incompletude alguma (com zero faltando, a igualdade continua válida: 0 == 0). Confirmar isso em vez de assumir.
5. Acrescentar ao teste um comentário apontando que 1,000 em C1 **não** prova atribuição correta de página, e que a prova está no teste multi-página da issue #2. Sem isso o número vira propaganda.

**Where to look**
- `api/cmd/provenance-eval/main_test.go:20-90`

**Acceptance criteria**
- [ ] `go test ./cmd/provenance-eval/...` passa.
- [ ] Nenhuma constante foi copiada da saída da ferramenta: o comentário mostra a derivação.
- [ ] O teste continua travando valor exato, sem faixa de tolerância além do epsilon de 0,001 já existente.

**Dependencies** #3

**Out of scope**
Não alterar o corpus C1.

---

## Issue #5 — Reescrever a subseção de auditabilidade na cópia do artigo

**Context**
O artigo afirma que `TableSection`, `MetadataPair` e `DocumentChunk` não têm coluna de página. Duas dessas três afirmações mudaram. Publicar o número novo sem explicar a causa parece maquiagem de métrica; publicar 1,000 sem ressalva é pior ainda, porque C1 tem uma página só.

**What to do**
1. Duplicar `docs/article/article-overleaf.md` como `docs/article/article-overleaf-v2.md`. O original não é editado em nenhuma hipótese.
2. Na cópia, reescrever a subseção `\subsection{Auditabilidade do Índice}` (por volta da linha 648) com: os três números lado a lado (0,329 do critério antigo; o efeito da página no chunk; o valor por escopo), a definição de escopo, e a causa de cada mudança.
3. Declarar na mesma subseção a limitação: C1 é texto de página única, então a TPC alta ali prova que o mecanismo carrega a página, não que a página é a certa; a evidência de atribuição correta é o teste multi-página.
4. Corrigir a frase que atribui a causa a `TableSection`.

**Where to look**
- `docs/article/article-overleaf.md:648-685`
- Números finais: saída de `cmd/provenance-eval` depois da issue #4.

**Acceptance criteria**
- [ ] `docs/article/article-overleaf-v2.md` existe e o original está intocado (`git diff --stat` não mostra `article-overleaf.md`).
- [ ] A subseção cita os três números e explica cada um.
- [ ] A limitação de página única está escrita no corpo da subseção, não em nota de rodapé.
- [ ] Nenhum número no texto diverge da saída da ferramenta.

**Dependencies** #4

**Out of scope**
Não reescrever outras seções do artigo nesta issue.

---

## Mapa de dependências

```
#1 escopo no pacote provenance ─┐
                                ├─> #3 avaliador ──> #4 constantes ──> #5 artigo
#2 página no chunk ─────────────┘
```
