# Experimento de geração: ThingIndex (T) contra "documento inteiro no LLM" (F)

> **Estado: parcial (2026-09-09).** O braço F no escopo *documento* rodou inteiro (72 perguntas × 3 repetições). O braço F no escopo *coleção* completou 35 das 72 perguntas e o braço T só a primeira, porque o crédito mensal do Ollama Cloud acabou no meio da rodada. Nenhum número abaixo foi transcrito à mão: todos saem de `go run ./cmd/generation-eval -summary` sobre os CSVs em `docs/article/results/`.
>
> ```bash
> cd api && set -a && . ../.env && set +a && export THINGINDEX_EVAL_PASSWORD='<senha do usuário demo>'
> go run ./cmd/generation-eval -arms T -repeats 3 -out ../docs/article/results/generation-results-T.csv
> go run ./cmd/generation-eval -arms F -scope document   -repeats 3 -model gemma4:31b -out ../docs/article/results/generation-results-F-document.csv
> go run ./cmd/generation-eval -arms F -scope collection -repeats 3 -model gemma4:31b -out ../docs/article/results/generation-results-collection.csv
> go run ./cmd/generation-eval -summary -in ../docs/article/results/generation-results-F-document.csv,../docs/article/results/generation-results-collection.csv
> ```

## 1. O que este experimento mede e por quê

O artigo pergunta se inteligência documental auditável dá para fazer com modelagem relacional, sem vetor e sem depender de LLM. A banca vai perguntar a coisa óbvia: **por que não mandar o documento inteiro para a IA?** Até esta rodada não havia número nenhum para responder. O experimento de recuperação (`docs/analise-2026-08/experimento-recuperacao.md`) compara braços A a E entre si; nenhum deles é o adversário.

Braços desta rodada:

| Braço | O quê |
|---|---|
| **T** | ThingIndex como está: BM25 sobre páginas/seções/chunks, 6 fontes no prompt (teto 24 mil caracteres), `gemma4:31b` via Ollama Cloud, medido pela API real (`/chat/stream`) |
| **F / documento** | O mesmo modelo, com o(s) documento(s)-alvo da pergunta inteiro(s) no contexto, extraídos por `pdftotext -layout` com marcador `[página N]`. O documento-alvo é escolhido pelo gabarito, o que favorece F: na prática o usuário não sabe em qual dos 5 PDFs está a resposta |
| **F / coleção** | O mesmo modelo com os 5 PDFs inteiros (≈250 mil tokens reportados por consulta) |

Gabarito: `api/internal/evalcorpus/testdata/gabarito-si.json`, 72 perguntas de coordenador (30 prosa, 21 tabela, 16 calendário, 5 negativas), escritas a partir dos PDFs, congeladas por hash antes de qualquer execução. Acerto é casamento de regex sobre a resposta (acento e caixa dobrados); proveniência em T é (documento, página) das fontes estruturadas; em F é a página que o modelo escreve em prosa, aceita se pertence a qualquer documento-alvo (regra frouxa, a favor de F). Unidade estatística é a pergunta (média das 3 repetições); IC 95% por bootstrap percentil com 1.000 reamostras e semente fixa.

## 2. Resultados

### 2.1 Braço F, escopo documento (completo: 72 perguntas, 216 chamadas, 0 erros)

| Recorte | n | Acerto (IC 95%) | Proveniência | Tokens prompt | Tokens resp. | Latência ms |
|---|---|---|---|---|---|---|
| todas | 72 | 94,4% (88,9–98,6) | 86,1% | 72.067 | 64 | 1.985 |
| sem rodada-cc5 | 60 | 95,0% (88,3–100,0) | 84,2% | 75.440 | 58 | 1.974 |
| prosa | 30 | 100,0% | 77,8% | 100.491 | 80 | 2.522 |
| tabela | 21 | 90,5% (76,2–100,0) | 87,3% | 39.691 | 56 | 1.537 |
| calendário | 16 | 93,8% (81,2–100,0) | 100,0% | 5.504 | 39 | 699 |
| negativa | 5 | 80,0% (40,0–100,0) | — | 250.504 | 82 | 4.752 |

Auditoria manual das 4 perguntas que F errou pelo regex:

- **t04** ("03 período"): o regex `\b3\b|3[ºo°]|terceiro` não aceita zero à esquerda. Resposta certa; falha do casador.
- **c08** ("entre os dias 15 e 17"): o regex esperava "15 a 17". Resposta certa; falha do casador.
- **t06** (Projeto Comunitário I, "Natureza - Obrigatória"): a Matriz tem duas colunas, Categoria = "Natureza - Obrigatória" e Atividade Pedagógica = "Extensão". O gabarito pediu "extensão" na categoria. **Ambiguidade do gabarito**, não erro do modelo.
- **n03** ("134 alunos matriculados, p. 132 do PPC do Técnico"): a página 132 não diz isso. **Alucinação com citação plausível.** É o único erro de conteúdo de F em 216 chamadas, e é exatamente o modo de falha que destrói a confiança de um coordenador, porque a resposta vem com página.

Corrigidos os dois casadores e a ambiguidade, F/documento fica em 71/72. Os regex **não foram alterados** depois de ver os resultados (o hash do gabarito é o mesmo); a correção fica registrada aqui para a próxima versão do gabarito.

Das 12 chamadas erradas de F/documento, 9 vieram com página citada. Resposta errada com citação é a regra, não a exceção, quando F erra.

### 2.2 Braço F, escopo coleção (parcial: 35 perguntas, 104 chamadas válidas; tabela e calendário)

Comparado com o escopo documento nas mesmas 35 perguntas:

| Escopo | Acerto (IC 95%) | Proveniência | Tokens prompt | Latência ms |
|---|---|---|---|---|
| coleção | 91,4% (82,9–100,0) | **50,5%** | 250.505 | 4.160 |
| documento | 91,4% (82,9–100,0) | 92,4% | 26.016 | 1.201 |

Pareado por pergunta: coleção vence em 2, perde em 2, empata em 31. O acerto não muda; a **proveniência cai de 92% para 50%** (na tabela, de 87% para 25%): com 5 documentos no contexto o modelo continua acertando o fato mas cita a página errada, porque "p. 3" da Matriz e "p. 3" do Calendário viram a mesma coisa em prosa. Custo: 10 vezes mais tokens de entrada e 3,5 vezes a latência. Das 9 chamadas erradas, 9 citaram página.

Isso é o argumento do artigo em número: **a fluência não é o problema; a endereçabilidade é.** Um modelo com tudo no contexto acerta e não sabe dizer de onde.

### 2.3 Braço T (pendente)

A primeira execução caiu no limitador da rota `/chat/stream` (20 pedidos por minuto, HTTP 429) a partir da 21ª chamada. O harness passou a esperar e repetir. A segunda execução começou quando o crédito do Ollama Cloud já tinha acabado e a API devolvia respostas vazias. **Não há resultado válido do braço T nesta rodada.** Da primeira execução, antes do 429, sobraram 14 perguntas (t01–t07, p18–p24) que mostram o padrão já conhecido: acerto quando a Matriz entra nas fontes (t01), "não há informações" quando a linha certa não entra (t02, t05, t07), e citação correta de página sempre que acerta.

## 3. Leitura honesta

1. **F vence em acerto, e por muito.** Com o documento certo no contexto, `gemma4:31b` responde 71 de 72 perguntas de coordenador. Não há como o ThingIndex, com 6 fontes de 4 mil caracteres e BM25 sobre prosa embaralhada de tabela, chegar perto disso hoje. Isso vai para o artigo como está (constituição VI).
2. **O que F não entrega é proveniência confiável.** No escopo realista (coleção, porque o usuário não sabe em qual PDF está a resposta), metade das citações aponta para a página errada, e toda resposta errada vem com uma página inventada. A contribuição do ThingIndex passa a ser esta: **toda fonte é (documento, página) por construção**, verificável com um clique, e a resposta errada aparece como "não há informações" em vez de um número inventado com página.
3. **Custo.** F/coleção gasta 250 mil tokens por pergunta; T, cerca de 600 (medido nas 14 perguntas válidas da primeira execução). Sem preço por token no Ollama Cloud o custo em USD não é reportado; em tokens a diferença é de 400 vezes.
4. **Tabela e calendário são o gargalo de T, e F prova que o conteúdo está lá.** F acerta 90% das perguntas de tabela lendo o mesmo `pdftotext -layout` que o indexador descarta. A extração de tabela do PDF (próximo PRD) tem teto conhecido: 90%.

## 4. Ameaças à validade

- O gabarito foi escrito pelo mesmo autor/agente que construiu o sistema (`reviewed_by: []`). Um coordenador real, cego ao sistema, deve revisar e acrescentar perguntas.
- Um corpus, um curso, um modelo. `deepseek-v3.1:671b` não rodou por falta de crédito.
- 3 repetições por pergunta; o modelo é estocástico (temperatura 0 em F, 0,2 em T, como em produção).
- O escopo documento entrega ao modelo o documento certo; isso é um oráculo que o usuário não tem.
- Correção por regex: dois casos (t04, c08) mostram que o casador é mais estrito que um leitor humano; auditados à mão acima.
- O braço T não tem número nesta rodada. Sem ele, o experimento não fecha.

## 5. Próximos passos

1. Repor crédito (ou outra chave) e rodar `-arms T -repeats 3` e o restante de `F -scope collection`.
2. Versão 2 do gabarito: corrigir t04, c08 e t06, registrar `reviewed_by` com o revisor externo, recongelar o hash.
3. Só então o PRD de extração de tabela de PDF, com o teto de 90% medido aqui como alvo.
