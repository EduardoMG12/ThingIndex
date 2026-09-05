# Plano 001 — Recuperação em nível de nó e Workspace de proveniência

Companion de `spec.md`. Aqui está **como** fazer, na ordem, com o que medir antes
de escrever código.

---

## Achado que define o desenho de P1

Medi o corpus antes de planejar. Dois fatos:

**1. O corpus sustenta granularidade fina.** Os 5 documentos têm 4-5 seções
numeradas cada (`1 RELATÓRIO`, `2 FUNDAMENTAÇÃO`, `3 DISPOSITIVO`, ...), ou seja
**~24 unidades** contra 5 documentos hoje. Isso quebra os tetos aritméticos: com
24 alvos possíveis, `R@5 = 1,000` deixa de ser automático e `P@3` deixa de ter
teto de 0,375.

**2. O gabarito atual é `relevance: {document_id: grau}` e
`expected_outline_titles` está `null`.** Esse campo foi **removido** do corpus
porque era circular — o rótulo era cópia literal da saída do próprio extrator
estrutural (registrado em `internal/evalcorpus/corpus.go:8`).

> **Consequência inegociável:** o gabarito de nó **não pode** ser ancorado no
> título que o extrator produz. Se eu escrever `relevance: {"acordao#2
> FUNDAMENTAÇÃO": 3}`, basta renomear a heurística de heading para a métrica
> desabar — medindo autoconsistência, não recuperação. Seria reintroduzir
> exatamente o bug que o projeto já pagou para remover.

**Desenho correto:** o corpus declara suas próprias seções, à mão, com id estável
e âncora textual. O harness casa a saída do extrator contra essas âncoras. O
rótulo passa a depender do texto-fonte, não do extrator.

```jsonc
// corpus-c1.json — cada documento ganha:
"sections": [
  { "id": "acordao-trabalhista#s3",      // id estável, autoral, nunca gerado
    "anchor": "3 DISPOSITIVO",           // ancora textual para casar a extracao
    "char_start": 612 }                  // fallback deterministico
]
// e cada consulta passa a poder graduar em nivel de secao:
"relevance_sections": { "acordao-trabalhista#s3": 3 }
```

Se a extração não casar com nenhuma âncora, o harness **aborta** (constituição,
princípio VI: produzir evidência inválida é pior que não produzir nenhuma).

---

## Ordem de execução

> **Status em 2026-09-05:** Fases 0 a 3 **concluídas** (issues #16, #17 e #18
> fechadas). Resultados medidos registrados abaixo e no handover §10. A Fase 4
> (Workspace) segue não iniciada, por depender de decisão do autor.
>
> | Fase | Entregue | Achado principal |
> |---|---|---|
> | 0 | ✅ | 26 nós brutos / 19 seções anotadas; baseline `P@1` de 0,200 → 0,053 |
> | 1 | ✅ | Queda literal→paráfrase do relacional: `−0,429` → **`−0,857`** |
> | 2 | ✅ | TPC bimodal: 1,000 onde há `page_number`, 0,000 onde não há |
> | 3 | ✅ | Existe limiar: falso positivo 1,000 → 0,500, Youden `J = 0,673` |
> | 4 | ⬜ | não iniciada |

### Fase 0 — Antes de qualquer código (0,5 dia)

1. Enviar o artigo atual ao orientador. É o único item que não depende de mim e
   pode reordenar tudo.
2. ~~Rodar o indexador e contar os nós reais.~~ **FEITO — 2026-09-04.**

#### Resultado da medição (risco principal eliminado)

Rodei `parsePlainTextStructure` (a função que o corpus de texto puro usa) sobre
os 5 documentos:

| Documento | Nós brutos do extrator | Seções de conteúdo anotadas |
|---|---|---|
| acordao-trabalhista | 5 | 3 |
| edital-pregao | 5 | 4 |
| contrato-sla | 6 | 4 |
| politica-privacidade | 5 | 4 |
| manual-onboarding | 5 | 4 |
| **Total** | **26** | **19** |

> **Os dois números medem coisas diferentes e ambos importam.** O extrator devolve
> 26 nós porque conta os títulos de capa (`ACÓRDÃO`, `RECURSO ORDINÁRIO
> 0001234-56.2025.5.09.0011`, `CONTRATO DE PRESTAÇÃO...`, `ANEXO II — NÍVEIS DE
> SERVIÇO`). Eles são estrutura legítima, mas **não são unidades que respondem a
> uma consulta** — ninguém pergunta algo cuja resposta seja "ACÓRDÃO". O gabarito
> anota as **19 seções de conteúdo**, e é esse o universo de recuperação.
>
> O número travado em teste (`secoesC1 = 19`) é o do gabarito, porque é ele que
> entra nas métricas do artigo.

**O corpus sustenta P1 sem escrever documento novo** — o risco de "outline raso
demais" está descartado por medição, não por estimativa.

Impacto nos tetos que hoje invalidam a tabela de resultados, calculado sobre as
19 seções anotadas:

| Métrica | Hoje (unidade = documento) | Com seção |
|---|---|---|
| `R@5` | **1,000 sempre** — recupera 5 de 5, é aritmética, não resultado | 5 de 19: volta a discriminar |
| `P@3` teto médio | **0,375**, não declarado na tabela | **0,458** |
| `P@1` aleatório | **0,200** | **0,053** |
| Consultas com >1 alvo | 2/16 | **6/16** |

A baseline aleatória cair de 0,200 para 0,053 é o número que mais importa: hoje
acertar por acaso é fácil, e é por isso que os braços empatam. Com seções, o
experimento passa a ter poder de discriminação real.

> A extração já produz hierarquia (`L1`/`L2` em `acordao-trabalhista` e
> `contrato-sla`), então `ParentID` tem conteúdo verdadeiro para a sidebar de P4.



### Fase 1 — P1: unidade de recuperação (3-5 dias)

| # | Tarefa | Teste que trava |
|---|---|---|
| 1.1 | Adicionar `sections[]` ao corpus com id estável e âncora | Teste que falha se algum `id` de seção não for único ou não casar no texto |
| 1.2 | Guarda de circularidade estendida para seções | Teste que falha se um rótulo de seção for cópia literal do título extraído |
| 1.3 | `relevance_sections` nas consultas positivas | Teste que falha se uma consulta tiver relevância de documento sem a de seção correspondente |
| 1.4 | Braço relacional ranqueia seções (`cmd/retrieval-eval/main.go:199-208`) | Teste de determinismo: duas execuções, mesmo ranking, desempate explícito |
| 1.5 | Harness aborta com gabarito antigo | Teste que passa gabarito só-documento e espera erro, não número |
| 1.6 | Tabelas publicam o teto de cada métrica | Teste que compara o teto calculado com o publicado |

**Ponto de integração:** `api/cmd/retrieval-eval/main.go:199-208`. Os braços
recebem `docs` e devolvem ranking de documento; passam a receber unidades
recuperáveis e devolver ranking de nós. O resto do harness (métricas, Wilcoxon,
Cliff's δ, CSV) não muda de forma.

**Degradação (RNF-1):** se um documento não tiver seções declaradas, ele entra
como unidade única — o experimento não quebra, só perde granularidade naquele
item, e isso é reportado.

### Fase 2 — P2: auditabilidade no artigo (1-2 dias)

`internal/provenance` já tem TPC e TCV com testes. Falta trazê-las ao texto.

| # | Tarefa | Teste que trava |
|---|---|---|
| 2.1 | Comando que emite a tabela de proveniência em `.tex` | Teste que compara a tabela gerada com a execução real |
| 2.2 | TPC **por tipo estrutural**, não só média | Teste com fixture onde um tipo tem proveniência incompleta; a média esconderia, a quebra por tipo não |
| 2.3 | Guarda de CI: número no artigo == número medido | CI falha se divergirem (constituição, princípio III) |

### Fase 3 — P3: limiar medido (2-3 dias)

| # | Tarefa | Teste que trava |
|---|---|---|
| 3.1 | Diagnóstico de separabilidade score×relevância | Teste com distribuições sintéticas separáveis e sobrepostas; o script tem de acertar as duas |
| 3.2 | Reportar o limiar ótimo **ou** a ausência dele | Teste que garante que sobreposição total devolve "sem limiar", não um número |
| 3.3 | Braço com limiar mede falso positivo | Teste que prova que abaixo do limiar o resultado é vazio, não o melhor mesmo assim |

> Este é o item que ataca o falso positivo 1,000 — a limitação mais grave medida
> no trabalho. Mesmo um resultado negativo ("não existe limiar separador neste
> corpus") é publicável e honesto.

### Fase 4 — P4: Workspace (5-8 dias, só se as fases 1-3 estiverem fechadas)

| # | Tarefa | Teste que trava |
|---|---|---|
| 4.1 | Rota que devolve nós + resumos de uma coleção | Teste de isolamento: usuário B não alcança nós do usuário A |
| 4.2 | Canvas com cartões e posição persistida | Teste de componente: arrastar e recarregar mantém posição |
| 4.3 | Sidebar com outline e resumos existentes | Teste que prova zero chamada de LLM no caminho |
| 4.4 | Conexões derivadas sob demanda | Teste que prova que nada é persistido: duas chamadas, mesmo resultado, zero linhas novas no banco |
| 4.5 | "Nenhuma conexão acima do limiar" | Teste que força scores baixos e espera lista vazia, não o melhor item |
| 4.6 | E2E no Chromium | Login → upload → indexar → abrir workspace → focar nó → pedir conexões |

**Biblioteca:** React Flow. Não há lib de canvas no repositório hoje (verificado
em `package.json`). É dependência nova e precisa ser justificada no PR.

---

## Disciplina exigida pela constituição

- **Princípio III:** toda propriedade afirmada no artigo tem teste que a trava.
  Nenhuma tarefa acima entra sem o teste da coluna direita.
- **Princípio VI:** número no texto vem de execução real. Nenhuma tabela é
  transcrita à mão; todas são geradas do CSV.
- **Fluxo:** commit isolado para mudança de fixture/corpus/gabarito, para ser
  revertível sozinho.
- **Disciplina de código:** zero `TODO`/`FIXME`. Pendência vira issue.

## Ordem de corte se o prazo apertar

P4 → P3 → P2. **P1 não é cortável**: é a correção do defeito que a banca vai
achar sozinha.

## Verificação final da feature

- [ ] O experimento ranqueia nós e o CSV tem uma linha por (consulta, braço, nó)
- [ ] O gabarito de seção não é cópia de saída do extrator (guarda automática)
- [ ] Tetos de métrica publicados ao lado dos valores
- [ ] TPC/TCV no artigo, por tipo estrutural, com guarda de CI
- [ ] Limiar reportado como medido, ou ausência declarada
- [ ] Conexões calculadas na hora, nada persistido, provado por teste
- [ ] E2E no Chromium cobre o caminho completo
- [ ] `go test -race ./...` e a suíte de frontend continuam verdes
