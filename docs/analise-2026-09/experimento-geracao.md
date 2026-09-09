# Experimento de geração: ThingIndex (T) contra "documento inteiro no LLM" (F)

> **Estado: braços T e F/documento completos (2026-09-09, 72 perguntas × 3 repetições cada).** F/coleção foi completado com a chave reserva (seção 7); o modelo maior está na seção 8. Nenhum número abaixo foi transcrito à mão: todos saem de `go run ./cmd/generation-eval -summary` sobre os CSVs em `docs/article/results/`.
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

## 3. Leitura honesta (atualizada na rodada 3)

1. **Em acerto, ThingIndex e "documento inteiro no LLM" empatam.** T 92,1%, F/documento 94,4%, F/coleção 95,8%; os intervalos se sobrepõem e a diferença pareada T × F/documento (+0,023) tem IC que cruza o zero. A rodada 1 tinha dito "F vence, e por muito": era verdade antes das tabelas virarem fonte (as perguntas de tabela estavam quase todas erradas em T).
2. **Em proveniência, T é estrutural e F é prosa.** T: toda fonte é (documento, página), 100% certa em tabela. F/coleção: 64% das páginas citadas certas, 25% em tabela, e toda resposta errada vem com uma página. O único erro de conteúdo de F/documento (n03) é uma alucinação com citação plausível.
3. **Em custo, 18 a 62 vezes menos tokens de entrada por pergunta** (4 mil contra 72 mil com o documento certo, 250 mil com a coleção). Em USD não há número: o Ollama Cloud é assinatura.
4. **F/documento recebe um oráculo** (o documento certo escolhido pelo gabarito) que o usuário não tem. F/coleção é o cenário real e é o mais caro.
5. **Tabela e calendário deixaram de ser o gargalo de T** depois da extração de tabela (seção 5). O que sobra em T é vocabulário na prosa (plural × singular, letra solta como "conceito D").

## 4. Ameaças à validade

- O gabarito foi escrito pelo mesmo autor/agente que construiu o sistema (`reviewed_by: []`). Um coordenador real, cego ao sistema, deve revisar e acrescentar perguntas.
- Um corpus, um curso, um modelo. `deepseek-v3.1:671b` não rodou por falta de crédito.
- 3 repetições por pergunta; o modelo é estocástico (temperatura 0 em F, 0,2 em T, como em produção).
- O escopo documento entrega ao modelo o documento certo; isso é um oráculo que o usuário não tem.
- Correção por regex: dois casos (t04, c08) mostram que o casador é mais estrito que um leitor humano; auditados à mão acima.
- O braço T não tem número nesta rodada. Sem ele, o experimento não fecha.

## 5. Rodada 2 (2026-09-09): tabelas de PDF como fonte, medidas sem LLM

O braço F provou que o conteúdo das tabelas está no `pdftotext`; o ThingIndex não o via porque o pipeline de PDF nunca emitia tabela. Entrou a detecção geométrica de tabelas pelas caixas de palavra (PRD `docs/prds/tabelas-de-pdf-como-fonte.md`), uma `table_section` por tabela por página, e cada linha virou unidade de recuperação e fonte de chat com (documento, página, tabela, linha).

Sem crédito de LLM, o que dá para medir é a **recuperação isolada da geração**: a API emite as fontes antes de chamar o modelo, então `cmd/generation-eval -arms T` registra se a página-alvo do gabarito está entre as fontes mesmo com o provedor fora. Isso responde à cobrança dos revisores do conselho ("Recall@k da recuperação isolado") e é determinístico (constituição II).

| Estrato | n | Antes | Depois |
|---|---|---|---|
| todas | 72 | 73,1% | **89,6%** |
| tabela | 21 | 42,9% | **100,0%** |
| calendário | 16 | 100,0% | 100,0% |
| prosa | 30 | 80,0% | 76,7% |

Doze perguntas de tabela passaram a ter a linha certa entre as fontes; a única perda (p01) traz a linha "TOTAL 3000" da Matriz p. 3, que responde à pergunta numa página que o gabarito não listou. A primeira tentativa, com linhas ocupando 3 das 6 vagas de fonte, derrubou a prosa para 63%; a versão final deixa as linhas fora das vagas de página (custam 100 caracteres contra 4.000) e corta linha abaixo de metade do score da melhor.

Com as caixas das tabelas em mãos, o sumário deixou de promover célula a título: Matriz 16 → 4 seções, PPC 564 → 492, PPC Técnico 511 → 495, Calendário 18 → 12. O Recall nas fontes foi medido de novo depois dessa limpeza e não mudou (89,6% no total, 100% em tabela): o ganho veio das linhas, não do sumário.

O que ainda não foi medido: se o modelo, recebendo a linha "Disciplina: SIS07302 - Estrutura de Dados I; Pré-requisito Trilha de Disciplina Hrs optativas: SIS07206" como primeira fonte, responde certo. Isso é a rodada 3, quando houver crédito. Confundidor: a reindexação rodou sem LLM, então os resumos e a árvore de seções caíram em fallback e os hits de seção mudaram entre antes e depois.

## 6. Rodada 3 (2026-09-09): o braço T medido, com as tabelas como fonte

Com a chave reserva, o braço T rodou inteiro: 72 perguntas × 3 repetições, `gemma4:31b`, ThingIndex com as linhas de tabela como fonte (seção 5). Três chamadas estouraram o timeout de 45 s da API e foram refeitas; nenhuma linha com erro ficou no CSV (`generation-results-T.csv`).

| Estrato | n | T: acerto (IC 95%) | F/documento: acerto (IC 95%) | T: proveniência | F: proveniência | T: tokens | F: tokens |
|---|---|---|---|---|---|---|---|
| todas | 72 | **92,1%** (85,2–97,7) | 94,4% (88,9–98,6) | 89,6% | 86,1% | 4.046 | 72.067 |
| prosa | 30 | 88,9% (76,7–98,9) | 100,0% | 76,7% | 77,8% | 3.462 | 100.491 |
| tabela | 21 | **95,2%** (85,7–100) | 90,5% (76,2–100) | **100,0%** | 87,3% | 3.551 | 39.691 |
| calendário | 16 | 91,7% (77,1–100) | 93,8% (81,2–100) | 100,0% | 100,0% | 5.817 | 5.504 |
| negativa | 5 | **100,0%** | 80,0% (40–100) | — | — | 3.956 | 250.504 |

Pareado por pergunta: F vence em 7, perde em 4, empata em 61; diferença média de acerto +0,023 a favor de F, **IC 95% de −0,056 a +0,106**, ou seja, indistinguível de zero com 72 perguntas.

Leitura:

1. **Em acerto, o ThingIndex empata com "mandar o documento certo inteiro para a IA"**, e vence na tabela (95% × 90%) e nas negativas (100% × 80%: F inventou "134 alunos matriculados" com página; T disse que não consta). Isso com 18 vezes menos tokens de entrada por pergunta (4 mil × 72 mil) e sem o oráculo: F recebeu o documento certo escolhido pelo gabarito; T procurou nos cinco.
2. **Em proveniência, T é estrutural**: toda fonte é (documento, página) por construção, verificável com um clique, e em tabela a página certa está entre as fontes em 100% das perguntas. A proveniência de F é a página que o modelo escreve em prosa, aceita se cair em qualquer documento-alvo (regra frouxa a favor de F).
3. **Onde T ainda erra.** Prosa: 4 perguntas em que a página-alvo não entra nas fontes (p06 "conceito D", p13 "estágio extracurricular", p24, p25), todas por vocabulário (plural × singular, "conceito D" com a letra solta). Tabela: t12 (Libras), o modelo leu o PPC p. 142 em vez da linha da Matriz que estava nas fontes. Calendário: c13 e c14 são o modelo escrevendo "do dia 12 ao dia 16" e "18 e 19" (o calendário diz 17, 18 e 19), casos em que o regex é estrito ou o modelo omitiu um dia.
4. **Antes desta rodada** (CC5, 11 perguntas escritas pelo autor, sem tabela como fonte): 8 certas, 2 parciais, 1 errada, com todas as de tabela erradas. O ganho veio da extração de tabela, não do modelo.

## 7. Braço F, escopo coleção, completo (72 perguntas)

Os 5 PDFs inteiros no contexto, ≈250 mil tokens por pergunta. As 35 primeiras perguntas têm 3 repetições (rodada 1) e as 37 restantes uma (rodada 3, para poupar crédito); 141 chamadas, sem erro. Arquivo `generation-results-F-collection.csv`.

| Estrato | n | Acerto (IC 95%) | Proveniência | Tokens prompt |
|---|---|---|---|---|
| todas | 72 | 95,8% (90,3–100) | **63,7%** | 250.505 |
| prosa | 30 | 100,0% | 76,7% | 250.504 |
| tabela | 21 | 95,2% (85,7–100) | **25,4%** | 250.505 |
| calendário | 16 | 87,5% (68,8–100) | 89,6% | 250.506 |
| negativa | 5 | 100,0% | — | 250.504 |

Este é o adversário realista, porque o usuário não sabe em qual PDF está a resposta: acerta tanto quanto T (95,8% × 92,1%, IC sobrepostos), gasta **62 vezes mais tokens por pergunta** (250 mil × 4 mil), e cita a página certa em 64% dos casos, 25% nas tabelas. Todas as 9 respostas erradas vieram com página citada. Para o coordenador, "acha o fato" é igual; "prova onde está" é o que separa os dois.

## 8. Rodada 4 (2026-09-09): corpus ampliado para 10 documentos e modelo maior

**Modelo maior no braço F.** `deepseek-v3.1:671b` foi aposentado pelo Ollama Cloud e os modelos pagos exigem assinatura; o maior disponível na conta é `gpt-oss:120b`. F/documento com ele, 1 repetição: **68,1%** de acerto (IC 56,9–79,2), proveniência 10,4%, negativas 0%, 6 erros (timeouts e "prompt is too long: 227.649, máximo 131.072" na coleção das negativas). Pior que o `gemma4:31b` nos dois braços: modelo maior não é modelo melhor para este uso, e o contexto de 131 mil tokens já não cabe a coleção. Arquivo `generation-results-F-document-gpt-oss-120b.csv`.

**Cinco documentos a mais.** Entraram na coleção de SI a Lei 11.788/2008 (estágio), a Resolução CNE/CES 5/2016 (DCN de Computação), a Resolução IFPR 50/2017 (avaliação) e os calendários de 2025 e 2024 (`examples/faculdade-novos/FONTES.md`). Os revisores do conselho tinham cobrado generalização; a pergunta aqui é se a recuperação degrada com mais documentos competindo pelas mesmas 6 fontes.

| Estrato | 5 documentos: página-alvo nas fontes / acerto (1 rep.) | 10 documentos: idem |
|---|---|---|
| prosa | 23/30 · 26/30 | 23/30 · 27/30 |
| tabela | 21/21 · 20/21 | 21/21 · 20/21 |
| calendário | 16/16 · 15/16 | 16/16 · 15/16 |
| negativa | — · 5/5 | — · 5/5 |

Nenhuma pergunta perdeu a página-alvo nem o acerto com o dobro de documentos, inclusive com dois calendários a mais (2024 e 2025) competindo com o de 2026: o ano na pergunta e no título separa as versões. Oito perguntas novas sobre os documentos novos, à mão: jornada de estágio pela Lei 11.788 (certa, p. 4 da lei), início do ano letivo de 2025 (certa, 20/02, calendário de 2025), fim de 2024 (certa, 13/12), conceito D e frequência pela Resolução 50/2017 (certas, mas citando o PPC do Técnico, que transcreve a resolução, em vez da própria), cursos abrangidos pela DCN (certa, pelo PPC), duração máxima na mesma concedente (errada: "não menciona"; o art. 11 da lei, na p. 5, não entrou nas fontes).

Duas coisas apareceram nessas oito: linhas do calendário entram como fonte em quase toda pergunta, porque uma linha curta com um único termo em comum ("Equivalência de Estágio") pontua alto no BM25; e a pergunta que nomeia o documento ("segundo a Lei 11.788", "pela Resolução 50/2017") não puxa as páginas daquele documento, só as de quem o cita. Correção medida na seção 9.

## 9. Rodada 4, correções tentadas no retrieval (medidas no gabarito, 10 documentos, 1 repetição)

| Mudança | Página-alvo nas fontes (prosa · tabela · calendário) | Acerto (prosa · tabela · calendário) | Decisão |
|---|---|---|---|
| base, 10 documentos | 23/30 · 21/21 · 16/16 | 27 · 20 · 15 | — |
| plural → singular no BM25 (5 documentos) | 24/30 · 19/21 · 16/16 | 26 · 19 · 14 | **descartada**: ganhou p06 e p13, perdeu t19, t20 e p21 |
| reforço de +300 por termo da pergunta no título do documento | 23/30 · 19/21 · 16/16 | 27 · 17 · 15 | **descartada**: "Sistemas de Informação" está em quase toda pergunta e no título do PPC; as tabelas do PPC passaram na frente da linha certa da Matriz (t02, t06, t07) |
| linha de tabela só é fonte com 2+ termos significativos da pergunta (token inteiro, sem pontuação e acento) | 23/30 · 21/21 · 16/16 | 27 · 20 · 15 | **mantida**: nada perdeu; perguntas de prosa com linha de calendário entre as fontes caíram de 5 para 3. A primeira versão perdia t02 e t07 porque "operacionais?" e "são" não eram normalizados antes de contar; o teste `TestTermosDaPerguntaSaoNormalizadosAntesDeContar` prende isso |

O que as duas rejeições ensinam: com 72 perguntas de gabarito e o CSV bruto, uma "melhoria óbvia" é testada em seis minutos e descartada com número, que é o oposto de ajustar no olho. As duas mudanças que sobreviveram nesta análise (tabelas como fonte, seção 5; token inteiro nas conexões) foram as que resolviam um erro estrutural, não as que mexiam no peso.

## 10. Próximos passos

1. Rodar F/documento em um segundo modelo pago (só há Ollama Cloud).
2. Versão 2 do gabarito: corrigir t04, c08 e t06, registrar `reviewed_by` com o revisor externo, recongelar o hash.
3. Só então o PRD de extração de tabela de PDF, com o teto de 90% medido aqui como alvo.
