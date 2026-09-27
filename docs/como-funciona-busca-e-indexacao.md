# Como funciona, exatamente: da indexação até a resposta

> Levantado lendo o código em 2026-09-19, com o número de linha de cada afirmação.
> Escrito porque "o sistema busca em SQL" é vago demais para defender numa banca,
> e porque a decisão sobre o enriquecimento depende de saber onde cada coisa entra.

## Parte 1: o que acontece quando um PDF é enviado

### 1.1 Extração

`api/internal/indexing/helpers.go:57` chama `pdftotext -bbox-layout`. A escolha
importa: o modo `-layout` antigo devolvia só texto, sem coordenada e sem fronteira
de página confiável. O modo com caixa entrega **página explícita e a caixa de cada
bloco e linha** já no espaço renderizado, com rotação e recorte aplicados.

PDF sem camada de texto utilizável cai no OCR por página, e a contagem de páginas
continua válida (`helpers.go:73-81`).

### 1.2 O que vira tabela

`api/internal/indexing/outline.go:36-56` monta, para cada documento:

| Tabela | O que guarda | Coordenada |
|---|---|---|
| `document_pages` | o texto da página inteira | número da página |
| `paragraph_nodes` | cada parágrafo | página, sequência, caixa, linhas |
| `document_outline_nodes` | a árvore de seções | página inicial e final |
| `table_sections` | cada tabela, uma por página | página |
| `document_chunks` | trechos para a busca | página (desde 19/09/2026) |
| `metadata_pairs` | estatísticas do documento | nenhuma, por natureza |

**Nada disso usa IA.** Verificado na prática: uma conta sem provedor nenhum indexou
a NR-6 e produziu exatamente os mesmos 229 parágrafos, 16 seções e 36 trechos que
a conta com IA. A diferença foi apenas os 12 resumos de página, que ficaram vazios.

### 1.3 O enriquecimento, que é opcional

Com um provedor de LLM configurado e ligado para indexação, o sistema gera:

- resumo de 25 a 60 palavras por página, em lotes de 6
- resumo por seção, em lotes de 12
- até 5 tópicos por página, em `document_pages.metadata.topics`
- sinopse do documento

Custo medido: cerca de 616 mil tokens de entrada e 101 mil de saída para 10
documentos (486 páginas).

## Parte 2: o que acontece quando alguém pergunta

Tudo em `buildCollectionRetrieval`, `api/internal/usecase/document_usecase.go:1959`.

### 2.1 A pergunta é tokenizada

`tokenizeRetrievalText` (`:2305`) dobra acento e caixa, tira pontuação das pontas,
e tem duas regras que vieram de falhas reais:

- **letra maiúscula sozinha conta como termo** ("conceito D", "TCC I"). Sem isso, a
  pergunta sobre o conceito D achava qualquer página com a palavra "conceito".
- letra minúscula sozinha continua fora (artigo, preposição).

`significantQueryTokens` (`:1877`) remove os conectivos. É essa lista que uma linha
de tabela precisa casar para virar fonte.

Antes do BM25, a consulta ainda ganha (`:1863`):
- variante numérica ("seven" também casa "7")
- **variante singular e plural** de cada termo, de modo que "estágio extracurricular"
  case "estágios extracurriculares"

Repare: a expansão acontece só na consulta, nunca no índice. Um termo a mais nunca
remove uma página, só soma quando a variante existe nela.

### 2.2 Cada tipo de estrutura é pontuado por BM25

| Estrutura | Texto que entra no BM25 | Multiplicador | Teto |
|---|---|---|---|
| Título do documento | o título | **× 100** | um por documento |
| Seção | título + `retrieval_text` | × 1 | 8 |
| Página | **`resumo + conteúdo`, colados** | × 2 | 8 |
| Trecho | título + conteúdo | × 2 | só pontua o documento |
| Linha de tabela | a linha com o cabeçalho colado em cada célula | **× 5** | 8 |

Duas regras de corte que vieram de medição:

- **Linha de tabela precisa casar pelo menos 2 termos significativos** (`:2110`).
  Sem isso, uma linha curta com um termo genérico pontuava altíssimo no BM25
  justamente por ser curta: "Descrição: Equivalência de Estágio" vencia qualquer
  pergunta que contivesse "estágio".
- **Reforço por título do documento foi testado e descartado** (`:1978-1981`):
  "Sistemas de Informação" está em quase toda pergunta e no título do PPC, e o
  reforço puxava as tabelas do PPC na frente da linha certa da Matriz. Perdeu 3
  perguntas de tabela e não ganhou nenhuma. O título só pontua o documento.

### 2.3 O documento é ranqueado pela melhor evidência, não pela soma

`keep(documentID, kind, score)` (`:1998`) guarda, para cada documento, **a melhor
pontuação de cada tipo**. A soma de todas as evidências foi descartada porque um
documento longo com muitas páginas fracas vencia o documento que tinha a tabela
certa.

Top 5 documentos, top 8 de cada tipo de trecho.

### 2.4 Só então o modelo entra

As fontes selecionadas viram o contexto do prompt. **Se não houver provedor de LLM,
o stream emite o evento de fontes normalmente e falha só na redação**, com a
mensagem "Nenhuma LLM está configurada para o chat". Verificado em 19/09/2026.

## Parte 3: onde isso deixa o enriquecimento

O resumo entra em um lugar só, e é o de menor alavancagem possível:

```go
pageTexts[i] = strings.Join([]string{page.Summary, page.Content}, " ")  // :2045
```

Três consequências, todas contra:

1. O resumo tem 25 a 60 palavras contra 300 a 700 da página. É cerca de 10% do texto.
2. O BM25 normaliza por comprimento, então **colar o resumo deixa a página mais longa
   e reduz levemente a própria pontuação dela**.
3. O resumo é escrito pela LLM no vocabulário do documento, então quase não traz
   termo novo.

O resumo de seção é pior ainda: `outlineNodeSearchText` (`:2297`) só o usa quando
`retrieval_text` está vazio, o que praticamente nunca acontece. Na prática, **o
resumo de seção é ignorado na busca**.

### A medição que fecha o caso

Para as 10 perguntas do gabarito de segurança do trabalho, contando quantos termos
significativos da pergunta aparecem no conteúdo da página e quantos aparecem no
resumo:

| Pergunta | Variante | No conteúdo | No resumo | **Só no resumo** |
|---|---|---|---|---|
| sg1 | literal | 1/6 | 0/6 | **0** |
| sg1 | paráfrase | 1/7 | 0/7 | **0** |
| sg2 | literal | 2/4 | 0/4 | **0** |
| sg2 | paráfrase | 0/5 | 0/5 | **0** |
| sg3 | literal | 4/4 | 4/4 | **0** |
| sg3 | paráfrase | 1/7 | 0/7 | **0** |
| sg4 | literal | 4/5 | 1/5 | **0** |
| sg4 | paráfrase | 3/8 | 0/8 | **0** |
| sg5 | literal | 4/5 | 0/5 | **0** |
| sg5 | paráfrase | 1/6 | 0/6 | **0** |

**Zero em todas.** O resumo não acrescenta um único termo que já não esteja na
página. Para um índice lexical, ele é redundância pura.

A causa está num exemplo só. Resumo gerado para a página 3 da NR-9:

> "define o nível de ação como metade dos limites de tolerância para agentes
> químicos e ruído"

Pergunta de uma pessoa real:

> "a partir de quanto de produto químico no ar a empresa já tem que tomar
> providência?"

Uma palavra em comum. O resumo foi escrito no jargão que a pessoa não usa, e é por
isso que ele não constrói ponte nenhuma.

## Parte 4: o que isso implica

O BM25 só consegue usar **vocabulário que não está no documento**. Segue que
enriquecimento só paga quando introduz termo ausente do original. Essa é a forma
geral do achado, e ela é testável: qualquer enriquecimento futuro pode ser avaliado
pela coluna "só no enriquecimento" antes mesmo de rodar o gabarito.

A correspondente na experiência do usuário é a queda medida de 0,875 para 0,625
quando a mesma necessidade é expressa com palavras de leigo.
