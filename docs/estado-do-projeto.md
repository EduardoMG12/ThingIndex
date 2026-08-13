# Estado do Projeto — ThingIndex

> Levantamento feito em 2026-08-12. Atualizado após os PRs #3 e #4.
> Ordenado por urgência. A última seção é tão importante quanto a primeira:
> ela diz o que **não** precisa de trabalho.

---

## Contexto rápido

O que foi consertado no PR #2 (para não reabrir o assunto):

| Bug | Onde | Efeito |
|---|---|---|
| Separação de páginas | `pdf_extract.go` | Todo PDF era indexado como se tivesse 1 página |
| OCR inalcançável | `helpers.go` | PDF 100% digitalizado nunca chegava ao fallback |
| Linha fragmentada | `helpers.go` | Fragmentos viravam heading fantasma no outline |
| HTML duplicado | `html.go` | `<li>` aninhado entrava duas vezes nos chunks |
| Ciclo de import | `processing` | Alvo `-tags=integration` não compilava |
| Fixture quebrada | corpus do benchmark | Palavras sobrepostas e texto fora da página |

Ganho colateral: cada bloco de PDF agora carrega sua **bounding box**, persistida
em `ParagraphNode.Metadata` (sem migration, a coluna já era `jsonb`).

---

## 1. Urgente

### ~~1.0 O repositório público não compilava~~ — RESOLVIDO (PR #3)

O achado mais grave de todos, e o mais silencioso. Um clone limpo do repositório
falhava no build:

```
internal/api/routes.go:35:  undefined: middleware.RequestContextMiddleware
internal/api/routes.go:49:  undefined: newUnavailableStorage
internal/api/routes.go:120: undefined: middleware.AuthMiddleware
```

**Causa:** `api/.gitignore` tinha o padrão solto `api` na seção de binários. A
intenção era ignorar o binário compilado `api/api`, mas sem a barra inicial o
padrão casa com *qualquer* diretório chamado `api` em qualquer nível — inclusive
o pacote de código `internal/api/`.

O efeito foi silencioso e por isso durou tanto: arquivos criados ali nunca
apareciam no `git status`, então nunca eram staged nem commitados. Cinco arquivos
existiam apenas na máquina local:

- `internal/api/middleware/auth.go` — **o middleware de JWT inteiro**
- `internal/api/middleware/auth_test.go`
- `internal/api/middleware/request_context.go`
- `internal/api/export.go`
- `internal/api/unavailable_storage.go`

Para um TCC com repositório público, isso é crítico: a banca pode clonar e nada
compila. Padrões agora ancorados (`/main`, `/api`, `/api-server`), o que preserva
o comportamento pretendido — `api/api`, 54 MB, segue ignorado — sem capturar
código-fonte. Verificado com clone limpo antes e depois.

> **Lição que vale além deste caso:** `.gitignore` sem barra inicial casa em
> qualquer profundidade. Para binário na raiz do módulo, sempre `/nome`.

### ~~1.1 Não existe CI~~ — RESOLVIDO (PR #3)

`.github/` continha apenas `prompts/`, nenhum workflow. Nada rodava teste em push
ou PR.

Não era hipótese: foi a causa-raiz direta do que encontramos. Um teste vermelho
(`TestIndexingBenchmarkProviderBuildsPublicReport`) e um alvo de build que nem
compilava (`-tags=integration`) sobreviveram vários commits porque ninguém
percebeu.

Agora existe `.github/workflows/ci.yml`, com dois jobs em push na `main` e em
todo PR:

- **backend**: `gofmt` (falha se houver arquivo desformatado), `go vet`,
  `go vet -tags=integration` (só para garantir que o alvo compila) e `go test ./...`
- **frontend**: `lint`, `typecheck` e `test:unit`

O runner instala `poppler-utils` e `tesseract-ocr` de propósito: sem eles os
testes de PDF caem em *skip* e passam sem testar nada, que é o pior resultado
possível.

### ~~1.2 O pre-commit está desenhado para falhar~~ — RESOLVIDO (PR #3)

`scripts/pre-commit-checks.sh` lia `THINGINDEX_PRECOMMIT_REAL_LLM` do `.env`.
Com `true`, todo commit tentava conectar no Postgres em `127.0.0.1:5434` e fazer
chamadas **reais e pagas** de LLM.

Sem a stack Docker de pé, todo commit falhava, e o resultado prático era
previsível: `--no-verify` (foi o que aconteceu no PR #2) e o hook virava
decoração — pior que não ter hook, porque dá falsa sensação de proteção.

O smoke de LLM real saiu do hook e virou sob demanda:

```bash
./scripts/test.sh backend-real-llm
```

O `pre-commit` mantém apenas o que é rápido e offline. A flag foi removida de
`.env.example` e das três seções de documentação que ainda a citavam.

### 1.4 O artigo do TCC não tem backup nenhum — DECISÃO SUA

Mesma classe de bug do 1.0, em outro arquivo. O `.gitignore` da raiz continha a
linha `docs/`, que ignorava a pasta inteira. **Zero arquivos de `docs/` estavam
versionados.**

O que existe apenas no disco local, sem backup e sem histórico:

| Arquivo | Tamanho |
|---|---|
| `docs/article/article-overleaf.md` | 79 KB — **o artigo** |
| `docs/article/references.bib` | bibliografia |
| `docs/article/Apresentação de TCC ThingIndex Slides.pdf` | 5,9 MB — slides da defesa |
| `docs/analise-para-correcao/*.md` | análise do referencial teórico |
| `docs/presentation/apresentacao.md` | roteiro |

Isso também explica por que o README apontava para `docs/article/README.md` e
`docs/proposta-continuacao-tcc.md`: a documentação descrevia arquivos que nunca
entraram no repositório.

**Estado atual:** a pasta `docs/` foi destravada para documentação de engenharia
(este arquivo, por exemplo), mas os artefatos de pesquisa continuam ignorados de
propósito. Commitar o artigo num repositório **público** antes da defesa é
decisão de publicação do autor — pode esbarrar em regra institucional, em
política de publicação prévia, ou simplesmente não ser desejado ainda.

**As opções:**

1. **Versionar no repo público** — remova as linhas de `.gitignore`. Simples, mas
   o artigo fica visível para qualquer pessoa antes da defesa.
2. **Repositório privado separado** para os artefatos acadêmicos. Melhor
   isolamento, mas mais um lugar para manter.
3. **Backup fora do git** (Overleaf já é um, Drive, etc.). O nome
   `article-overleaf.md` sugere que talvez já exista cópia no Overleaf — se sim,
   confirme que está sincronizada, porque a cópia local está 100% desprotegida.

Enquanto nenhuma for escolhida, o risco continua: uma falha de disco custa o
artigo inteiro.

### ~~1.3 Páginas via OCR não têm âncora posicional~~ — RESOLVIDO (PR #5)

Foi o único pedaço que ficou de fora do PR #2. Blocos vindos do `tesseract`
entram sem `Box`, então o grifo posicional simplesmente não existe justamente em
documento digitalizado — o caso que mais importa para processo judicial.

**Resolvido por outro caminho:** em vez de extrair bbox do tesseract, o OCR
virou plugável e ganhou um provider baseado no `baidu/Unlimited-OCR`, que devolve
`<|det|>tipo [bbox]<|/det|>conteúdo` — texto e região juntos.

O `pdfOCRExtractor` agora devolve blocos em vez de texto puro. O tesseract
continua como fallback e entrega blocos sem `Box`; o Unlimited-OCR entrega
ancorados, convertidos de pixel da imagem para pontos PDF pelo DPI de
rasterização, no mesmo espaço das caixas do `-bbox-layout`.

**Requer GPU NVIDIA.** O modelo tem 3B params em BF16 (~6 GB só de pesos), então
sobe sob o profile `ocr` do compose e é opcional por construção — sem ele o
projeto continua funcionando com tesseract em qualquer máquina.

> **Pendente de conferência:** a escala das coordenadas (`THINGINDEX_OCR_BOX_SCALE`,
> padrão 1000 = normalizado) não pôde ser validada contra uma execução real do
> modelo, por falta de GPU disponível. Escala errada desloca o grifo sem gerar
> erro. Confira na primeira execução e ajuste a variável se necessário.

---

## 2. Médio prazo

### ~~2.1 `internal/repository` não tem um único teste~~ — CORREÇÃO + RESOLVIDO (PR #4)

**A primeira versão deste documento afirmava que `internal/repository` não tinha
nenhum teste. Isso estava errado.** O erro veio da forma de medir: o comando
usado contava apenas arquivos de teste que o Go compila por padrão, e os desse
pacote estão atrás de `//go:build integration`. Havia 8 testes — o maior arquivo
do pacote é justamente um deles.

O que já era coberto: outbox (claim de pendentes, incremento de tentativas),
auditoria de provider e documento com filtro e cursor, criação de documento com
registros derivados, retry de indexação e deleção em cascata por collection.

Mas duas lacunas reais existiam, e essas foram resolvidas:

**O código que sustenta a tese não era testado.**
`gorm_collection_context_repository.go` reconstrói o contexto de uma collection —
é a frase do artigo virada em SQL. Nenhum teste mencionava `CollectionContext`.
Agora há 6, cobrindo exatamente as propriedades afirmadas:

| Teste | Propriedade do artigo |
|---|---|
| `ReconstructsStableReadingOrder` | ordem de leitura sai por `sequence_order`, não por ordem de inserção |
| `GroupsNodesByDocumentBeforeSequence` | ordenação composta: documento antes de sequência |
| `RejectsCollectionOfAnotherUser` | escopo por usuário na collection |
| `ExcludesDocumentsOfAnotherUser` | isolamento transitivo nas tabelas derivadas |
| `ReconstructionIsFaithful` | sem perda nem duplicação (produto cartesiano) |
| `QueryCountDoesNotGrowWithDocuments` | **agregação linear: 9 consultas fixas** |

O último é o que rende número para o artigo: **9 consultas para 1 documento e 9
para 20**. É a diferença entre afirmar "agregação linear" e demonstrá-la. A
contagem tem limite superior travado numa constante, então acrescentar uma
consulta obriga a revisar o número citado no texto.

> Detalhe honesto: a primeira versão do teste esperava 8 consultas e falhou. São
> 9 — o `Preload("Category")` emite uma consulta extra que eu não tinha contado.
> O teste corrigiu a minha suposição, que é exatamente o serviço que ele presta.

**Os testes nunca rodavam.** Precisam de Postgres, e o CI só verificava se
compilavam. Agora há um job `backend-integration` com service container de
Postgres 16. Escopo proposital em `./internal/repository`: os outros pacotes com
tag `integration` também exigem MinIO, RabbitMQ ou LLM real.

**Bônus encontrado ao rodar a suíte inteira pela primeira vez:**
`migrateIntegrationModels` não criava a tabela `document_processing_attempts`,
presente no `AutoMigrate` de produção. Um dos 8 testes existentes falhava com
`relation "document_processing_attempts" does not exist` — vermelho que ninguém
via, porque a suíte não rodava sozinha. Corrigido, e as duas listas de migration
agora conferem.

### ~~2.2 `pkg/crypto` sem teste~~ — RESOLVIDO (PR #3)

O código já estava **correto** (ver seção 3); faltava a verificação. Agora há 6
testes cobrindo round-trip, chave mestra errada, texto cifrado adulterado
(autenticação do GCM), ciphertext menor que o nonce, e frescor do nonce.

O mais importante deles trava a propriedade de isolamento: uma credencial
cifrada para o usuário 1 **não** decifra no contexto do usuário 2, porque o
`userID` entra na derivação da chave.

### 2.3 Arquivos grandes demais

| Arquivo | Tamanho |
|---|---|
| `usecase/document_usecase.go` | 71 KB |
| `usecase/indexing_benchmark.go` | 60 KB |
| `usecase/provider_usecase.go` | 35 KB |
| `pkg/provider/factory.go` | 33 KB |

Não é bug e não urge. É risco de manutenção: arquivo desse tamanho esconde
duplicação e dificulta revisão. Só vale quebrar quando você já for mexer neles
por outro motivo — refatorar por refatorar não paga.

### 2.4 Pendências administrativas

- **Sem `LICENSE`.** O próprio README admite. Repo público de TCC sem licença
  fica juridicamente ambíguo para quem quiser citar ou reusar. **Pendente de
  decisão sua** — escolher licença é chamada do autor, não do agente. MIT e
  Apache-2.0 são as usuais para trabalho acadêmico que se quer citável.
- ~~**Remote com nome antigo.**~~ Resolvido: `origin` agora aponta para
  `ThingIndex` em vez de `LegisGraph-`.
- **Arquivo `1` na raiz.** 0 bytes, untracked, provável typo de redirecionamento
  (`> 1`). Deixado intocado de propósito — é seu, apague quando quiser.

---

## 3. Não mexer

Tão importante quanto a lista de problemas. Estas áreas foram olhadas e estão
boas — trabalho aqui é desperdício.

**`pkg/crypto/aes.go`** — AES-GCM com chave derivada por usuário via HMAC-SHA256
e nonce aleatório prefixado no ciphertext. Correto. Um ciphertext de um usuário
não decifra sob a chave de outro, porque o `userID` entra na derivação. Falta só
o teste (2.2), não a correção.

**Pipeline de indexação determinística** — acabou de ser consertado e cada bug
ganhou teste de regressão explícito. `pdf_bbox_test.go`, `pdf_scanned_test.go` e
`html_nested_test.go` travam o comportamento. Não remexer sem motivo novo.

**Frontend** — 50 arquivos de teste para 133 componentes, com vitest, playwright
e storybook configurados. Proporção saudável, bem acima do backend.

**Arquitetura relacional** — `categories → collections → documents →
paragraph_nodes / table_sections / metadata_pairs` é coerente e sustenta a tese.
Não há motivo técnico para trocar por grafo ou vetor dentro do escopo do TCC.

**Disciplina de código** — zero `TODO`, `FIXME` ou `HACK` em todo o código Go e
TypeScript. Raro, e vale preservar.

**`api/internal/api/middleware/auth.go`** — o middleware de JWT estava fora do
repositório, mas o código em si está correto e tem teste (`auth_test.go`, que
também estava escondido). O problema era de versionamento, não de implementação.

**`indexing_benchmark.go`** — 60 KB é muito, mas é instrumento de medida, não
código de produção. Baixo risco.

---

## 4. Ordem sugerida

| # | Item | Estado |
|---|---|---|
| 0 | Repo não compilava / gitignore (1.0) | ✅ PR #3 |
| 1 | CI (1.1) | ✅ PR #3 |
| 2 | Enxugar o pre-commit (1.2) | ✅ PR #3 |
| 3 | Teste de crypto (2.2) | ✅ PR #3 |
| 4 | Testes de `repository` (2.1) | ✅ PR #4 — 6 testes novos + Postgres no CI |
| 5 | **bbox no OCR (1.3)** | ⬜ próximo — destrava o grifo em documento digitalizado |
| 6 | `LICENSE` (2.4) | ⬜ depende de decisão sua |
| ! | **Backup do artigo (1.4)** | ⚠️ decisão sua, e é a mais urgente da lista |

Com o CI de pé, os itens 4 e 5 passam a ser protegidos automaticamente contra
regressão — que era o ponto de fazer 1 e 2 primeiro.

---

## Ressalva sobre os números do benchmark

As métricas saíram de `structure=60 / node_top1=66,7 / exact=81,8` para
`100 / 100 / 100`.

**Parte desse ganho vem da correção da fixture do corpus, não da qualidade do
pipeline.** A fixture `supplier-master-order-form.pdf` renderizava palavras
sobrepostas e perdia texto fora da margem da página.

Ao citar no artigo, a leitura honesta é *"corrigimos um artefato do corpus e uma
regressão de segmentação"*, não *"o pipeline melhorou 40 pontos"*. O commit da
fixture (`3dac860`) foi mantido isolado justamente para ser revertível sozinho e
auditável.
