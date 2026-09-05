# PRD 002 — Confiabilidade sob falha e cobertura das camadas críticas

**Status:** proposto
**Data:** 2026-09-05
**Autor:** levantado por medição contra o sistema em execução, não por inspeção de código

---

## 1. Contexto

A rodada anterior (spec 001) fechou as quatro fatias de recuperação em nível de
seção e entregou a aba Workspace. Depois disso, uma caçada dirigida encontrou
**oito defeitos reais**, todos corrigidos e provados por reversão:

| # | Defeito | Como foi encontrado |
|---|---|---|
| 1 | Provider de OCR remoto era código morto | leitura dirigida do construtor de produção |
| 2 | Página em voo sobrescrevia o filtro novo | leitura + instrumentação |
| 3 | Categoria propagava nome sem trim | leitura do caminho até o upload |
| 4 | API sem encerramento gracioso | comparação com os outros dois binários |
| 5 | Consumidor não reconectava ao broker | **queda real do RabbitMQ** |
| 6 | Publicador não reconectava ao broker | **queda real do RabbitMQ** |
| 7 | `/health` respondia 200 com o banco fora | **queda real do Postgres** |
| 8 | Corrida na criação de coleção (HTTP 500) | **20 uploads simultâneos** |

O padrão é claro e é a premissa deste PRD: **os quatro defeitos mais graves só
apareceram quando a infraestrutura foi derrubada ou o sistema foi posto sob
carga.** Nenhum teste unitário os teria encontrado, e nenhum deles era visível
por leitura de código.

## 2. Problema

Três lacunas ficaram evidentes ao corrigir os oito defeitos.

### 2.1 Os cenários que acharam os bugs não estão no CI

As quedas de broker, banco e storage foram feitas por script manual em `/tmp`.
Se alguém reintroduzir o `log.Fatalf` no consumidor, ou trocar o
`EnsureCollection` por `CreateCollection`, **nada acusa** — os testes de
integração atuais exercitam serviços saudáveis.

Existem testes unitários do supervisor e da corrida de coleção, e eles pegam a
regressão direta. O que não existe é a prova de que o sistema **inteiro** se
recupera: a diferença entre "o supervisor reconecta" e "o documento chega a
`INDEXED` depois da queda".

### 2.2 Quatro pacotes com cobertura próxima de zero

Medido com `go test -cover`:

| Pacote | Cobertura | Risco |
|---|---|---|
| `internal/dto` | 0,0% | conversão silenciosa errada entre API e domínio |
| `internal/repository` | 0,8% | camada que fala com o banco; a corrida de coleção morava aqui |
| `internal/api` | 15,7% | montagem de rotas e middlewares |
| `internal/pkg/provider` | 26,0% | credenciais, catálogo de modelos, chamadas pagas |

`internal/repository` com 0,8% é o número mais alarmante: **o bug #8 estava
exatamente ali**, e a suíte não tinha como vê-lo porque os testes do pacote
exigem Postgres e rodam só com a tag `integration`.

### 2.3 O caminho de PDF é o menos exercitado

Medido enviando três PDFs reais: os três indexam corretamente (4, 4 e 2 nós;
zero páginas vazias). Mas quase toda a caçada usou HTML, que é o caminho fácil
do extrator. Não há teste automatizado que envie um PDF de verdade pelo
pipeline completo e confira a estrutura resultante — e PDF é o formato que o
trabalho promete tratar.

## 3. Objetivo

Transformar em **guarda automática** os cenários que provaram encontrar
defeitos, e cobrir as camadas onde um erro passa silencioso.

Não é aumentar cobertura por métrica: é cobrir **onde já apareceu defeito**.

## 4. Fatias

### Fatia 1 — Testes de caos no CI (prioridade máxima)

Um job que derruba cada dependência com o sistema em uso e verifica a
recuperação completa, não apenas a reconexão.

Cenários, todos já validados manualmente nesta sessão:
- RabbitMQ cai com upload em andamento → documento chega a `INDEXED` após a volta
- Postgres cai → `/health` responde 503 e volta a 200 sozinho
- MinIO cai → download falha com erro claro e volta sem reiniciar a API
- 20 uploads simultâneos para a mesma coleção → 20 × 201, uma coleção

**Critério de aceite:** cada cenário falha se a correção correspondente for
revertida. Sem isso o job é decoração.

**Risco declarado:** teste de caos é lento (~4 min) e pode ficar intermitente em
runner compartilhado. Mitigação: job separado, não bloqueante no PR, mas
obrigatório antes do merge para `main`.

### Fatia 2 — Cobertura de `internal/repository` sem exigir Postgres

Hoje 0,8%, porque tudo depende de banco real. A fatia cobre o que é lógica pura:
construção de query, filtros por usuário, mapeamento de erro do driver para erro
de domínio.

**Critério de aceite:** ≥ 40% no pacote, com pelo menos um teste que falharia se
um filtro `user_id` fosse removido de uma query — o mesmo tipo de defeito que a
auditoria de IDOR procurou manualmente.

### Fatia 3 — Cobertura de `internal/dto`

0,0% hoje. É a fronteira entre API e domínio: um campo esquecido no mapeamento
não quebra build nem teste, só some da resposta. Foi exatamente isso que
aconteceu com o `node_id` do sumário na rodada anterior — o backend enviava, o
frontend descartava.

**Critério de aceite:** todo conversor `To*Response` tem teste que verifica os
campos que o frontend consome de fato.

### Fatia 4 — PDF real no pipeline automatizado

Um teste de integração que envia os PDFs de `testdata` pelo indexador completo e
confere estrutura, páginas e proveniência.

**Critério de aceite:** o teste falha se a extração devolver zero nós, se alguma
página vier vazia num PDF com camada de texto, ou se o número de páginas
divergir do que o `pdfinfo` reporta.

### Fatia 5 — Fluxo completo no navegador

Os 23 testes E2E atuais usam rotas mockadas. Falta o percurso real: login →
upload de PDF → esperar indexação → ver a estrutura → abrir o workspace → pedir
conexões.

**Critério de aceite:** roda contra a stack de verdade, sem mock, e falha se
qualquer etapa quebrar.

**Risco declarado:** depende da stack de pé e é lento. Fica em suíte separada,
marcada, fora do CI padrão.

## 5. Fora de escopo, com motivo

- **Corpus expandido para o experimento** — trabalho de anotação manual, não de
  código. Continua sendo a limitação declarada do artigo.
- **Paralelizar o consumidor** — o processamento sequencial é decisão
  registrada; mudar exigiria repensar ordenação e limites de provedor. Não há
  evidência de que a fila seja gargalo: 20 documentos indexaram em 5 s.
- **Cobrir `cmd/*`** — são binários de composição; o valor está nos pacotes que
  eles montam.
- **Métrica de custo agregada por documento** — a instrumentação existe
  (`tokenUsage`), a agregação não. Depende de decisão sobre corpus.

## 6. Como o sucesso é medido

| Indicador | Hoje | Meta |
|---|---|---|
| Cenários de caos no CI | 0 | 4 |
| Cobertura `internal/repository` | 0,8% | ≥ 40% |
| Cobertura `internal/dto` | 0,0% | ≥ 60% |
| Teste automatizado com PDF real | não existe | 1 suíte |
| E2E sem mock (stack real) | 0 | 1 percurso completo |

Cada meta vem acompanhada de prova por reversão: o teste tem que **falhar** com
o defeito reintroduzido. Número de cobertura sem poder de detecção é métrica
vazia — e esta sessão já teve quatro testes que passavam com o código errado.
