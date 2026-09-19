# Gabarito multidomínio: estratos declarados no arquivo

**Status:** proposto e implementado (2026-09-19), branch `feat/proveniencia-completa-e-multidominio`
**Data:** 2026-09-19
**Origem:** o autor pediu manter a proposta genérica ("serve para vários casos") e sustentar isso com medição em mais de um domínio. O bloqueio apareceu ao tentar escrever o segundo gabarito.
**Autor ausente:** decisões tomadas sem entrevista, marcadas como Assunção.

## Problema

`internal/evalcorpus/gabarito.go` fixa no pacote as regras de um gabarito específico:

```go
var GabaritoStrata = map[string]int{"prosa": 20, "tabela": 15, "calendario": 10, "negativa": 5}
const GabaritoMinQuestions = 50
```

São as regras do acervo de ensino. Um gabarito de legislação não tem estrato "calendário" e não precisa de 50 perguntas para discriminar; um de recursos humanos teria outros estratos ainda. Com a regra no pacote, o segundo domínio só entra de duas formas ruins: inventando perguntas de calendário sobre uma lei federal, ou afrouxando a constante e enfraquecendo a validação do gabarito que já existe.

O efeito prático é que a afirmação genérica do artigo ("plataforma de inteligência documental") hoje se apoia em medição de um domínio só, e a infraestrutura de avaliação impede que isso mude.

## Goals

Um gabarito declara os próprios estratos e o próprio mínimo de perguntas, e a validação passa a usar o que o arquivo declara. O gabarito de ensino continua válido sem nenhuma alteração no arquivo. Fica possível medir um segundo e um terceiro domínio com o mesmo harness.

Pronto quando: `gabarito-si.json` valida sem mudar uma linha; um gabarito novo com estratos próprios valida; um gabarito que declara estrato e não o cumpre falha com mensagem que nomeia o estrato.

## Requirements

- [x] `Gabarito` ganha `MinQuestions int` (`min_questions`) e `Strata map[string]int` (`strata`), ambos opcionais.
- [x] Ausentes, valem os padrões de hoje. O gabarito de ensino não muda.
- [x] `Validate` usa os valores efetivos, e a mensagem de erro de estrato desconhecido lista os estratos aceitos naquele gabarito.
- [x] Teste: gabarito com estratos próprios valida; estrato não declarado é recusado; estrato declarado e não cumprido é recusado com o nome do estrato.
- [x] Teste de regressão: o gabarito de ensino continua válido e continua exigindo 50 perguntas e os quatro estratos originais.

## Non-Goals

- Não alterar `gabarito-si.json`.
- Não mudar `cmd/generation-eval`: ele consome o gabarito pela mesma interface.
- Não remover a checagem de `frozen_sha256`. O congelamento das perguntas é o que impede ajustar o sistema e depois ajustar a pergunta.

## Approach

Métodos privados `strata()` e `minQuestions()` no `Gabarito` devolvem o declarado ou o padrão. `Validate` passa a chamá-los. A mudança é de cinco linhas na validação e duas no struct; o resto é teste.

## Assunções (revisar)

1. **Padrão silencioso em vez de obrigatório.** Um gabarito sem `strata` herda o do ensino. A alternativa era exigir declaração explícita em todo arquivo, o que quebraria o gabarito existente sem ganho. O risco é um gabarito novo esquecer de declarar e herdar estratos que não fazem sentido para ele; a validação então falha por estrato desconhecido, que é uma mensagem clara.
2. **O mínimo por estrato continua sendo piso, não cota exata.** Um gabarito pode ter mais perguntas que o mínimo declarado.

## Issues

**#1 — Estratos e mínimo declarados no arquivo do gabarito**
Onde: `api/internal/evalcorpus/gabarito.go`, `gabarito_test.go`.
Aceite: os cinco testes da lista de requisitos passam; `go test ./internal/evalcorpus/...` verde.
Dependências: nenhuma.

**#2 — Gabarito de licitações (segundo domínio)**
Onde: `api/internal/evalcorpus/testdata/gabarito-licitacoes.json`, apontando para `examples/licitacoes/`.
Conteúdo: perguntas escritas a partir dos PDF com `pdftotext -layout`, sem consultar o sistema, com documento e página verificados. Estratos próprios do domínio, incluindo um que o gabarito de ensino não tem: **divergente**, em que a mesma pergunta tem resposta diferente em cada lei, e acertar significa distinguir as duas.
Aceite: `LoadGabarito` valida o arquivo; toda página declarada existe no PDF; `frozen_sha256` confere.
Dependências: #1.
