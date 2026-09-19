# Pré-registro: perguntas geradas em linguagem de leigo no lugar do resumo

**Status:** pré-registrado em 2026-09-19, antes de qualquer implementação
**Branch:** `feat/proveniencia-completa-e-multidominio`

> Este documento existe para ser escrito **antes** da intervenção, e não depois.
> Ele fixa a hipótese, o prompt exato, os braços de comparação e o critério de
> sucesso. Sem isso, rodar três versões de prompt e reportar a melhor produziria
> um número favorável e sem valor, e ninguém conseguiria detectar isso depois.

## O que já está estabelecido (não é o que se vai testar)

1. O enriquecimento atual gera um resumo por página e **não serve para a
   recuperação**. Medição: para 10 perguntas, termos significativos que existem
   só no resumo e não no conteúdo da página: **zero em todas as dez**.
2. A causa aparente está no próprio prompt, que manda a LLM dizer o que a página
   contém "with the numbers, names, dates and rules present in it"
   (`api/internal/indexing/prompts.go:153`). Ele pede, literalmente, para
   reutilizar o vocabulário da página.
3. O sistema acerta 0,875 quando a pergunta reutiliza as palavras do documento e
   **0,625** quando a mesma necessidade é expressa com palavras de leigo.

## Hipótese

Se o enriquecimento produzir **perguntas que a página responde, escritas no
vocabulário de quem não leu o documento**, ele passa a introduzir no índice
termos ausentes do original, e a recuperação de perguntas parafraseadas melhora.

A forma geral por trás disso, estabelecida na análise do mecanismo: um índice
lexical só consegue usar vocabulário que não está no documento. Enriquecimento
que repete o original é redundância, independentemente da qualidade do texto.

## Predição, registrada antes de rodar

- **Se a hipótese estiver certa:** o acerto nas perguntas parafraseadas sobe, e o
  das literais fica aproximadamente igual.
- **Se subir nas duas igualmente:** o efeito é de indexação (mais texto no índice),
  não de ponte de vocabulário, e a hipótese específica está errada.
- **Se não subir em nenhuma:** segunda evidência independente de que enriquecimento
  por LLM não ajuda recuperação lexical. Resultado negativo, e vale como achado.

## O prompt, congelado

Substituto do prompt de resumo de página. **Rodará uma vez. Não haverá segunda
versão nesta medição.** Qualquer ajuste posterior exige um pré-registro novo e a
medição recomeça.

Prompt de sistema:

```
You generate one stage of a structured document indexing pipeline.
Return only valid JSON.
Do not wrap the response in markdown fences.
Write only questions that the supplied page text actually answers.
Write them in the everyday Portuguese of a person who has NOT read the document.
Do not reuse the technical vocabulary of the page when a common word exists.
```

Prompt de usuário (antes do bloco de contexto das páginas):

```
Task: write the questions each indexed page answers, for retrieval by everyday vocabulary.
Return JSON with field pages.
pages must be an array of objects with page_number and questions.
questions must be an array of 3 to 5 short questions in Portuguese, each one a question a non-specialist would actually type.
Each question must be answerable by the page text alone.
Prefer the everyday word over the technical term: write "quanto tempo" instead of "prazo decadencial", "quem paga" instead of "responsabilidade pelo custeio".
Do not number the questions and do not answer them.
```

O hash deste prompt está registrado em `enrichmentPromptVersion` e no teste que
trava o texto, de modo que alterá-lo quebra o build.

## Os três braços

O resumo antigo **não é apagado**: ele vira controle negativo, já medido.

| Braço | Texto que entra no BM25 da página |
|---|---|
| **C** (base) | só o conteúdo |
| **R** (controle) | conteúdo + resumo em jargão |
| **P** (tratamento) | conteúdo + perguntas em linguagem de leigo |

O braço R é o que torna o resultado interpretável com amostra pequena: se P sobe
e R não, o mecanismo fica isolado, porque a única diferença entre os dois é o
vocabulário do texto acrescentado.

## Os quatro números

Não se reporta uma média. Reporta-se:

| | Literal | Paráfrase |
|---|---|---|
| Antes (braço R, já medido) | 0,875 | 0,625 |
| Depois (braço P) | a medir | a medir |

As literais são o **controle contra a armadilha**: se subirem junto, não houve
ponte de vocabulário.

## Como o teste estatístico será feito

Os 16 pares não são 16 amostras independentes: é o mesmo item medido em duas
formulações, com o gabarito travado por hash **antes** desta ideia existir. A
análise é pareada, por teste de sinais sobre os pares discordantes, e não por
diferença de médias. Isso é mais sensível com o mesmo número de itens.

Com n = 16, nenhum resultado será apresentado como conclusivo.

## O que garante que não há vazamento

1. As 16 perguntas parafraseadas foram escritas em 2026-09-19 pela manhã, a partir
   dos PDF, e congeladas por `frozen_sha256` **antes** de esta hipótese existir.
   Nenhuma delas será alterada.
2. As perguntas geradas pela LLM **não entram em nenhum gabarito**. Elas existem
   apenas dentro do índice.
3. O prompt gerador está congelado acima e roda uma vez.

A ressalva que permanece e será declarada: a mesma família de modelo gera as
perguntas do índice e redige a resposta final. Isso não é corrigível nesta
medição e vai na seção de ameaças à validade.

## Procedimento

1. Coluna nova para as perguntas geradas, separada do resumo. Nada é sobrescrito.
2. Reindexação em **cópia** das coleções. As coleções atuais ficam intactas, para
   que uma falha não deixe o autor sem a bateria nova nem a antiga.
3. Uma execução da bateria nos três temas, nos três braços.
4. Tabela dos quatro números e teste de sinais.

## Afirmação máxima permitida na apresentação

> "O enriquecimento por resumo é inútil para a recuperação: zero termos exclusivos
> em dez perguntas. Trocando o resumo por perguntas em linguagem de leigo, o acerto
> nas dezesseis perguntas parafraseadas pré-registradas foi de 0,625 para X, com as
> literais como controle. São dezesseis pares, hash anterior à ideia, uma repetição,
> e a mesma família de modelo nas duas pontas. É indício direcional e ainda
> inconclusivo; resolver isso exige gabarito maior."

Qualquer afirmação além desta, nesta amostra, é indefensável.
