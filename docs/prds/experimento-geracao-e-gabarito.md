# Braço F "documento inteiro no LLM" e gabarito de 50 perguntas

**Status:** em execução (2026-09-09), branch `feat/experimento-geracao` a partir de `feat/chat-contexto-e-custo` (PR #60, sem merge)
**Data:** 2026-09-09
**Origem:** veredito do conselho (llm-council) sobre "como tornar o sistema demonstravelmente melhor que mandar tudo para uma IA". O adversário do artigo nunca foi medido, e as 11 perguntas usadas até aqui foram escritas por quem conhece o sistema. Antes de mexer em tabela ou conexão, o experimento precisa saber contra o que compete.
**Base:** `docs/analise-2026-08/experimento-recuperacao.md` (braços A–E de recuperação sobre o corpus hermético C1), `docs/analise-2026-08/gap-artigo.md` (lacuna 1: sem baseline comparativa), `docs/prds/chat-contexto-e-custo.md` (as 11 perguntas e o uso por resposta).

## Goals

(1) Um gabarito de 50 perguntas de coordenador sobre a coleção de SI (PPC 2019, Calendário 2026, Matriz 2023, Ementas 2023, PPC do Técnico), cada uma com documento, páginas-alvo e padrões de resposta verificáveis por máquina, escrito a partir dos PDFs e não da saída do sistema, congelado por hash antes de qualquer ajuste. (2) Um experimento de geração ponta a ponta com dois braços sobre esse gabarito: **T**, o ThingIndex como está (BM25, 6 fontes, LLM), medido pela API real; e **F**, o mesmo modelo com o documento inteiro no contexto, sem recuperação. Cada pergunta roda 3 vezes por braço. (3) Números honestos: tokens reportados pelo provedor (não estimados), custo em USD só onde há preço, truncamento e erro registrados como resultado, intervalo de confiança por bootstrap.

Pronto quando: `go test ./internal/evalcorpus/...` valida o gabarito (50 entradas, estratos, páginas dentro do documento, padrões compilam); `go run ./cmd/generation-eval` produz `docs/article/results/generation-results.csv` com uma linha por (pergunta, braço, repetição) e um resumo por braço e por estrato; `docs/analise-2026-09/experimento-geracao.md` registra os números da primeira execução e o que eles dizem sobre a tese.

## Requirements

Gabarito
- [ ] `api/internal/evalcorpus/testdata/gabarito-si.json`: `version`, `frozen_sha256` (do campo `questions` serializado), `documents[]` (`id`, `file`, `pages`), `questions[]` com `id`, `text`, `stratum` ∈ {`prosa`, `tabela`, `calendario`, `negativa`}, `document_ids[]`, `pages[]`, `answer_patterns[]` (regex, casamento sem acento e sem caixa), `expected` (texto humano), `origin` (`pdf` ou `rodada-cc5`).
- [ ] Estratos mínimos: 15 `tabela` (matriz, quadros do PPC), 10 `calendario`, 20 `prosa`, 5 `negativa` (a resposta não está no acervo; `pages` vazio; padrão de resposta é a recusa: "não consta|não há|não encontr").
- [ ] Teste em Go: 50+ entradas, ids únicos, estratos nos mínimos, cada `pages[]` dentro de `documents[].pages`, cada regex compila, `frozen_sha256` confere. Mudar uma pergunta exige recalcular o hash e anotar em Discoveries.
- [ ] As 11 perguntas da rodada CC5 entram com `origin: rodada-cc5`; a métrica principal do artigo é reportada também sem elas (foram usadas para ajustar o sistema).

Uso de tokens
- [ ] `ollamaNativeLLMClient` passa a ler `prompt_eval_count` e `eval_count` da resposta do `/api/chat` e a implementar `TokenUsageReporter`. Teste com servidor falso. Sem isso o braço F reportaria estimativa de runas/4, e o truncamento de contexto ficaria invisível.

Harness `cmd/generation-eval`
- [ ] Braço T: login na API (`-api`, `-email`, `-password`), cria conversa na coleção (`-collection`), `POST /chat/stream`, consome SSE até `done`; guarda resposta, páginas das fontes, `usage` e latência.
- [ ] Braço F: para cada pergunta, texto dos `document_ids` do gabarito extraído com `pdftotext -layout` página a página, com marcador `[página N]`; prompt de sistema pede resposta em português citando páginas como "p. N"; provedor via `provider.NewLLMProvider` com `THINGINDEX_TEST_LLM_*` (modelo por `-model`). Variante `-scope collection` manda os 5 documentos; erro ou `prompt_tokens` menor que o enviado registra `truncated=true`.
- [ ] 3 repetições por (pergunta, braço), `-repeats`; temperatura igual à produção.
- [ ] CSV: `question_id, stratum, origin, arm, scope, repeat, model, correct, cited_pages, provenance_ok, retrieval_hit, prompt_tokens, completion_tokens, usage_reported, latency_ms, cost_usd, truncated, error, answer`.
- [ ] Correção: algum `answer_pattern` casa na resposta (dobra de acento e caixa). Proveniência: alguma página citada ∈ `pages[]` do gabarito (T: páginas das fontes; F: "p. N" extraído do texto). `retrieval_hit`: só T, alguma fonte ∈ `pages[]`. Negativa: correta quando casa o padrão de recusa e não cita página.
- [ ] Resumo (`-summary`): por braço e por estrato, acerto, proveniência, `retrieval_hit`, média de tokens, latência, custo; IC 95% por bootstrap percentil (1.000 reamostras, semente fixa) sobre perguntas; separa "com CC5" e "sem CC5".
- [ ] Custo em USD só com `-input-price`/`-output-price` (USD por milhão), rotulado no resumo como preço de referência; Ollama Cloud é assinatura e não tem preço por token.
- [ ] Unitários: casamento de padrão, extração de "p. N", cálculo de proveniência, bootstrap determinístico. Nada de rede em teste.

Relatório
- [ ] `docs/analise-2026-09/experimento-geracao.md`: comandos, modelo, tabela por braço e estrato com IC, casos em que F truncou, leitura honesta (se F vence em acerto, dizer; comparar custo por consulta e proveniência), e as ameaças à validade (gabarito escrito pelo autor, um corpus, 3 repetições).

## Non-Goals

- Extrair tabelas de PDF ou mudar o chat, o retrieval ou as conexões: PRD seguinte, depois dos números.
- Braço vetorial em produção; o braço denso segue só no experimento de recuperação.
- Card de custo na interface.
- Tocar no corpus C1 e no `cmd/retrieval-eval`.
- Rodar o harness em CI: ele depende de rede, de provedor e da API no ar.

## Constraints

- O experimento de recuperação já usa as letras A a E (E = RRF denso). O braço novo é **F**, e o ThingIndex ponta a ponta é **T**.
- Coleção de SI (id 86 no demo): PPC 209 p. (≈48,5 mil tokens estimados por runas/4), Técnico 159 p. (≈45,4 mil), Ementas 66 p. (≈11,6 mil), Calendário 5 p., Matriz 3 p. Um documento cabe no contexto dos modelos do Ollama Cloud; a coleção inteira (≈108 mil estimados, mais em tokens reais) pode não caber. Não caber é resultado, não impedimento.
- Só há chave do Ollama Cloud (`THINGINDEX_TEST_LLM_*`, modelos `gemma4:31b` e `deepseek-v3.1:671b`). Não há chave OpenAI/Anthropic. O braço F roda nos mesmos modelos que T.
- `domain.LLMService.GenerateText(ctx, system, user)` e `TokenUsageReporter` já existem; o cliente OpenAI-compatível reporta uso, o nativo do Ollama ainda não.
- SSE do chat: eventos `stage`, `source`, `delta`, `done` (com `usage`), `error`; fontes trazem `page` e `document_id`.
- Constituição VI: nenhum número no relatório é transcrito à mão; o resumo sai do CSV.

## Approach

1. Gabarito primeiro, lendo os PDFs (`pdftotext -layout`), sem consultar o sistema. Perguntas como um coordenador faz: pré-requisito, período, carga horária, prazo, feriado, vagas, regra de TCC e estágio, laboratório. Cinco negativas sobre o que o acervo não tem (bolsa, valor de mensalidade, nome do reitor atual).
2. `ollamaNativeLLMClient` reporta uso.
3. `cmd/generation-eval` com dois braços e CSV; funções puras testadas.
4. Rodar `-repeats 3` em `gemma4:31b` (o modelo do demo) e, se o crédito permitir, `deepseek-v3.1:671b`; escrever o relatório.

## Open Questions

- **Revisão cega do gabarito** (decisão do autor): as 50 perguntas serão escritas por quem construiu o sistema. A ameaça à validade fica registrada; o ideal é um coordenador ou professor revisar e acrescentar perguntas sem ver o sistema. O arquivo tem `reviewed_by: []` para registrar isso.
- **Modelo pago externo** (bloqueado): sem chave OpenAI/Anthropic. Se o autor fornecer, o harness roda F no mesmo comando via `THINGINDEX_TEST_LLM_PROVIDER=OPENAI`.
- **Empilhamento de PRs** (decisão do autor): este PR depende do #60 (uso por resposta no `done`). Sem merge dos #55 e #60, a base do PR é `feat/chat-contexto-e-custo`.

## Ambiguity Report (grill-me, modo spec, 2026-09-09)

```
Ambiguity Report:
  Goals:        0.0   ✓ clear
  Acceptance:   0.25  ✓ clear (correcao por regex e verificavel; a leitura "F vence ou nao" e registro, nao criterio)
  Boundaries:   0.0   ✓ clear
  Alternatives: 0.25  ✓ clear (HTTP x usecase direto para T: HTTP mede o produto real; LLM-juiz x regex: regex e deterministico e auditavel no CSV)
  Assumptions:  0.25  ✓ clear (verificado: letras A-E ocupadas, GenerateText/TokenUsageReporter, SSE done com usage, tokens por documento no banco; nao verificado: janela de contexto dos modelos do Ollama Cloud, tratada como resultado)
  ──────────────────────────────
  Aggregate:    0.15  ✓ below threshold (0.2 spec)
```

## Discoveries

**Rodada 1 (2026-09-09), antes dos números**

- O gabarito saiu com 72 perguntas (30 prosa, 21 tabela, 16 calendário, 5 negativas), 12 delas marcadas `rodada-cc5`. O hash foi recalculado uma vez antes de qualquer execução: o padrão de recusa das negativas ganhou as formas em inglês ("no information", "does not"), porque no smoke o braço T respondeu em inglês.
- O braço T respondia em inglês porque a API decide o idioma pelo header `Accept-Language`, que o frontend manda e o harness não mandava. Corrigido no harness; a instrução de idioma no prompt não bastava sozinha.
- O cliente Ollama nativo não lia `prompt_eval_count`/`eval_count`; agora lê (issue #63). Com isso o braço F tem tokens reportados, não estimados.
- No smoke, `gemma4:31b` no Ollama Cloud aceitou um prompt de 250.503 tokens reportados (a coleção inteira, 5 PDFs) sem erro nem truncamento detectável. O escopo `collection` é viável e entra na rodada.
- O braço T cria uma conversa por pergunta e a apaga ao fim, para não poluir o usuário demo.
- Perguntas negativas no escopo `document` recebem a coleção inteira (não há documento-alvo); registrado no relatório.

**Rodada 1, números (2026-09-09)**

- F/documento completo: 94,4% de acerto (IC 88,9–98,6), proveniência por página 86,1%; auditado à mão, 71/72 (dois casadores estritos, uma ambiguidade do gabarito em t06, uma alucinação com citação plausível em n03). F/coleção, 35 perguntas: mesmo acerto, proveniência de 50,5%, 250 mil tokens por consulta. Relatório em `docs/analise-2026-09/experimento-geracao.md`.
- A rota `/chat/stream` tem limitador de 20 pedidos por minuto; a primeira execução do braço T caiu em 429 na 21ª chamada. O harness espera 30 s e repete (até 10 vezes) e não conta a espera na latência.
- O crédito mensal do Ollama Cloud acabou no meio de F/coleção (a partir de c14) e antes da segunda execução de T. O harness agora interrompe o lote ao ver "usage limit" e trata resposta vazia da API como erro, não como "errada". **T fica pendente até haver crédito.**
- Os CSVs válidos estão versionados em `docs/article/results/` (`generation-results-F-document.csv`, `generation-results-collection.csv`, este com as linhas de erro mantidas para registro).

**Rodada 3 e 4 (2026-09-09), com a chave reserva**

- Braço T fechado (72 × 3): 92,6% de acerto (gabarito v2), proveniência 91%, 4 mil tokens por pergunta; F/documento 98,6%; F/coleção 98,6% com proveniência 64%. Diferença pareada T × F +0,060 (IC +0,000 a +0,130). Modelo maior disponível (`gpt-oss:120b`) foi pior (68%); `deepseek-v3.1:671b` foi aposentado pelo provedor e os maiores exigem assinatura.
- Gabarito v2: t04, c08, c13 com regex tolerantes; t06 aceita as duas colunas; p01 inclui Matriz p. 3. Hash recongelado. Os CSVs foram reavaliados com `cmd/generation-eval -rescore` (4 linhas de T, 9 de F/documento e 6 de F/coleção mudaram de veredito), sem rerodar modelo; o relatório traz v1 e v2.
- Corpus dobrado (10 documentos): nenhuma pergunta perdeu página-alvo nem acerto. Três correções de retrieval medidas: plural→singular e reforço por título descartadas; linha só com 2+ termos mantida.
- Bug encontrado por uma pergunta que falhava (p24): o texto da página não continha os títulos, então a identificação do PPC ("Eixo Tecnológico / Informação e Comunicação") nunca entrava no BM25 nem no prompt. Corrigido; o controle híbrido do benchmark público subiu de 90,9% para 100% por isso, e o teste registra o motivo.
- **Estado final (2026-09-09):** T com 10 documentos, títulos na página e enriquecimento ligado, 3 repetições: 96,3% de acerto (IC 91,7–100), tabela 100%, negativas 100%, proveniência 92,5%, 4,4 mil tokens por pergunta. F/documento 98,6%; diferença pareada +0,023 com IC de −0,028 a +0,074. Relatório, seção 10.
