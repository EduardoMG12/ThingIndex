# ThingIndex Constitution

ThingIndex é uma plataforma open source de inteligência documental construída
sobre modelagem relacional. Este documento fixa os princípios inegociáveis do
projeto. Ele existe porque o ThingIndex é, ao mesmo tempo, um artefato de
engenharia e o objeto de um trabalho de conclusão de curso: cada decisão técnica
precisa ser defensável diante de uma banca, não apenas funcional.

## Core Principles

### I. Auditabilidade sobre conveniência (NÃO NEGOCIÁVEL)

Todo dado recuperado deve ser rastreável até sua origem: documento, página,
posição sequencial e tipo de estrutura. Proveniência não é uma camada adicional —
é uma constraint de esquema.

- Nenhuma estrutura extraída pode ser persistida sem chave estrangeira para o
  documento de origem e sem posição de ordenação explícita.
- É proibido representar conteúdo estrutural como JSONB monolítico ou coluna
  polimórfica. Cada tipo de dado tem sua própria tabela tipada.
- Uma resposta de chat sem citação rastreável é um bug, não uma limitação.

Justificativa: é a hipótese central da pesquisa. Quebrar isso invalida o trabalho.

### II. Recuperação determinística como camada primária

O caminho primário de recuperação de contexto é SQL determinístico sobre chaves
estrangeiras. Embeddings, vetores e grafos são permitidos apenas como trilha
comparativa ou enriquecimento opcional — nunca como dependência do caminho
principal.

- O sistema deve funcionar por completo sem nenhum provedor de LLM configurado.
- O sistema deve funcionar por completo sem pgvector, sem banco vetorial e sem
  chamada de rede a serviço externo de embeddings.
- A mesma entrada produz a mesma saída. Não-determinismo no pipeline de
  indexação é bug.

Justificativa: "sem depender de bancos vetoriais como camada primária" é
literalmente a pergunta de pesquisa. Um atalho aqui responde outra pergunta.

### III. Test-First para propriedades afirmadas no artigo (NÃO NEGOCIÁVEL)

Toda propriedade afirmada no artigo precisa de um teste automatizado que a trave.
Afirmação sem teste é opinião.

- "Agregação linear" exige teste que conte consultas e falhe se o número crescer.
- "Isolamento por usuário" exige teste que prove que o usuário B não lê dado do
  usuário A — em cada repositório, não só em um.
- Todo bug corrigido entra com teste de regressão explícito nomeando o bug.
- Números citados no texto (contagens, métricas, limites) devem ser derivados de
  execução real, e o teste deve travar a constante para que alterá-la obrigue a
  revisar o artigo.

### IV. O repositório é a fonte da verdade

O que não está versionado não existe. Aprendido da forma difícil: um `.gitignore`
com padrão solto escondeu o middleware de JWT inteiro e a pasta `docs/` completa,
incluindo o artigo, sem backup.

- Um clone limpo deve compilar, rodar os testes e subir o ambiente. Isso é
  verificado, não presumido.
- Padrões de `.gitignore` para artefatos na raiz do módulo são sempre ancorados
  (`/api`, nunca `api`).
- O artigo e os artefatos da defesa são versionados junto do código, incluindo o
  PDF compilado de referência.

### V. CI que falha de verdade

Um teste que não roda é pior que um teste ausente: dá falsa segurança. Um teste
que passa por *skip* silencioso é a pior variante.

- Todo push e todo PR roda backend (gofmt, vet, vet -tags=integration, testes) e
  frontend (lint, typecheck, unit, e2e).
- Dependências que causariam *skip* (poppler, tesseract, Postgres) são instaladas
  no runner de propósito. Skip silencioso em teste de PDF é falha de CI.
- Hooks de pre-commit contêm apenas o que é rápido e offline. Nada que exija
  Docker de pé, rede ou chamada paga de LLM — isso força `--no-verify` e o hook
  vira decoração.

### VI. Honestidade nos números

O projeto reporta o que mediu, não o que gostaria de ter medido.

- Ganho vindo de correção de fixture é reportado como correção de fixture, não
  como melhoria de pipeline.
- Métrica heurística é rotulada como heurística. Estimativa de custo e de tokens
  é rotulada como estimativa quando o provedor não devolve uso real.
- Benchmark pequeno e determinístico é apresentado como baseline versionada, não
  como afirmação de acurácia geral.
- Um resultado negativo publicado vale mais que um resultado positivo inflado.

## Restrições Técnicas

- **Persistência**: PostgreSQL é o banco primário. Integridade referencial por
  chaves estrangeiras e consistência transacional (ACID) são obrigatórias.
- **Isolamento multi-tenant**: toda consulta que toca dado de usuário é escopada
  por `user_id`. Escopo transitivo (documento → coleção → usuário) conta e
  precisa de teste.
- **Credenciais**: credenciais de provedor são cifradas em repouso com chave
  derivada por usuário. Um ciphertext de um usuário não decifra sob a chave de
  outro.
- **Desacoplamento de provedores**: trocar storage (MinIO → S3 → GCS) ou LLM não
  pode exigir alteração na lógica de coleção ou de recuperação de contexto.
- **Resiliência do pipeline**: padrão outbox na mesma transação da escrita,
  retry com atraso e dead-letter queue. Falha de processamento não bloqueia
  novos uploads.
- **Reprodutibilidade**: ambiente completo sobe via Docker Compose com
  healthchecks, em Linux, macOS e Windows (WSL). Componentes que exigem hardware
  específico (ex.: OCR com GPU) ficam atrás de profile opcional, e o sistema
  degrada com elegância sem eles.
- **Internacionalização**: mensagens de API e interface em PT-BR e EN-US.

## Fluxo de Trabalho

- **Branches e PRs**: trabalho entra por PR. CI verde é pré-requisito de merge.
- **Commits atômicos e revertíveis**: mudança que altera dado de referência
  (fixtures, corpus, gabarito) vai em commit isolado, para ser auditável e
  revertível sozinha.
- **Disciplina de código**: zero `TODO`, `FIXME` ou `HACK` no código Go e
  TypeScript. Trabalho pendente vira issue, não comentário.
- **Refatoração oportunista**: arquivos grandes demais só são quebrados quando
  já houver motivo para mexer neles. Refatorar por refatorar não paga.
- **Documentação do estado**: `docs/estado-do-projeto.md` registra o que foi
  resolvido, o que está pendente e — igualmente importante — o que foi analisado
  e não precisa de trabalho.

## Governance

Esta constituição prevalece sobre preferência pessoal e sobre conveniência de
prazo. Em conflito entre entregar rápido e manter auditabilidade, determinismo
ou honestidade nos números, a constituição vence.

- Todo PR verifica conformidade com os princípios acima.
- Complexidade adicional exige justificativa explícita. Na dúvida, a opção mais
  simples e mais auditável ganha.
- Emenda a este documento exige registro do que mudou, por quê, e qual o impacto
  nas afirmações já feitas no artigo. Se uma emenda invalida um número citado no
  texto, atualizar o texto faz parte da emenda.
- Princípios marcados NÃO NEGOCIÁVEL não podem ser flexibilizados por prazo. Se
  não der tempo, o escopo diminui — o princípio não.

**Version**: 1.0.0 | **Ratified**: 2026-08-17 | **Last Amended**: 2026-08-17
