# ThingIndex - Backlog Reorganizado e Amadurecido

## Objetivo deste arquivo

- Separar com mais rigor o que ja existe, o que esta parcial e o que realmente falta.
- Evitar backlog inflado por itens que ja estao funcionais no codigo.
- Priorizar lacunas que fecham o produto, fortalecem o valor academico do TCC e aumentam o valor open source.
- Registrar proximos passos tecnicos de forma que faca sentido tanto para implementacao quanto para defesa tecnica.

## Painel Executivo

**Contagem atual do backlog:**
- 32 tasks catalogadas ao todo.
- 28 tasks ja concluidas no escopo atual: `A0`, `A1`, `A2`, `A3`, `A4`, `B1`, `B2`, `B3`, `B4`, `B5`, `C0`, `C1`, `C2`, `C3`, `D1`, `D2`, `D3`, `D4`, `D5`, `D6`, `E2`, `E3`, `E4`, `F1`, `F2`, `F4`, `G1`, `G2`.
- 4 tasks continuam abertas entre nao implementado, ideia ou questao estrategica.
- 0 tasks sao realmente de alta necessidade agora.
- 0 tasks sao importantes, mas nao urgentes.
- 4 tasks abertas sao mais de roadmap, expansao, manutencao ou valor academico.

**Leitura rapida:**
- Nem todas as tasks sao ultra necessarias.
- As frentes de consumo open source e onboarding deixaram de ser lacuna estrutural e sairam da fila principal.
- O foco do produto, o pacote academico inicial e a baseline objetiva de indexacao deixaram de ser lacunas estruturais.
- A leitura de custos deixou de ser lacuna estrutural e o backlog mais util agora comeca por portabilidade e manutencao.
- As tasks de IA mais ambiciosas, novos storages, CLI, refinos visuais e material academico agregam valor, mas nao bloqueiam o produto principal.

**Legenda usada abaixo:**
- `Necessidade Alta`: reduz risco, corrige percepcao enganosa, fecha comportamento essencial ou endurece o produto.
- `Necessidade Media`: melhora qualidade, DX, UX, operacao ou narrativa tecnica sem bloquear o uso principal.
- `Necessidade Baixa`: roadmap, expansao, polimento ou valor academico adicional.
- `Tamanho Pequeno`: cabe em rodada curta.
- `Tamanho Medio`: envolve mais de uma camada, mas segue bem delimitado.
- `Tamanho Grande`: depende de desenho, dados, bateria de testes, varias telas ou varias integracoes.

### Tabela-resumo das tasks abertas

| ID | Status | Necessidade | Tamanho | Leitura curta |
| --- | --- | --- | --- | --- |
| E1 | NAO IMPLEMENTADO | Baixa | Grande | Implementar storages alem de MinIO/S3-compatible |
| E5 | NAO IMPLEMENTADO | Baixa | Grande | Criar CLI para importar indices em banco externo |
| F3 | IDEIA CONTINUA | Baixa | Pequeno | Limpar boilerplate e naming excessivo |
| G3 | BAIXA PRIORIDADE | Baixa | Pequeno | Fazer refinos visuais menores |

### Tarefas ja concluidas e que nao devem poluir a fila principal

| ID | Tema | Situacao |
| --- | --- | --- |
| A0 | Bateria de testes automatizada | Concluido no escopo atual |
| A1 | Exclusao confiavel de documentos | Concluido no escopo atual |
| A2 | Download explicito do original | Concluido no escopo atual |
| A3 | Documento orfao com UX operacional | Concluido no escopo atual |
| A4 | Confirmacoes destrutivas restantes | Concluido no escopo atual |
| B1 | Retry, fila e DLQ operacionais | Concluido no escopo atual |
| B2 | Trilhas de auditoria consultaveis | Concluido no escopo atual |
| B3 | Healthcheck e readiness de providers | Concluido no escopo atual |
| B4 | Rastreabilidade do bug de upload percebido | Concluido no escopo atual |
| B5 | Cobertura de testes para fluxos criticos | Concluido no escopo atual |
| C0 | Integridade automatica antes de commit | Concluido no escopo atual |
| C1 | Revisao objetiva de seguranca e abuso | Concluido no escopo atual |
| C2 | Honestidade tecnica do frontend | Concluido no escopo atual |
| C3 | Internacionalizacao restante | Concluido no escopo atual |
| D1 | Acuracia de indexacao com avaliacao real | Concluido no escopo atual |
| D2 | Observabilidade operacional mais madura | Concluido no escopo atual |
| D3 | Feedback mais claro na UI sobre configuracao ativa de LLM | Concluido no escopo atual |
| D4 | Legibilidade e experiencia do chat | Concluido no escopo atual |
| D5 | Pagina de custo e leitura de pricing | Concluido no escopo atual |
| D6 | Pagina de monitoramento documental | Concluido no escopo atual |
| E2 | Revisao real do suporte aos outros providers de LLM | Concluido no escopo atual |
| E3 | Consumo open source da API e do projeto | Concluido no escopo atual |
| E4 | Onboarding para agentes e automacao | Concluido no escopo atual |
| F1 | Docs principais e secundarias alinhadas ao runtime atual | Concluido no escopo atual |
| F2 | Separacao entre funcional, parcial e roadmap | Concluido no escopo atual |
| F4 | Storybook estruturado no frontend | Concluido no escopo atual |
| G1 | Material academico derivado inicial | Concluido no escopo atual |
| G2 | Proposta de valor e foco do produto validados | Concluido no escopo atual |

## 1. Prioridade Alta - O Que Fecha o Produto

### [A0] Bateria de testes automatizada para frontend e backend

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Objetivo:**
- Fazer com que a validacao principal do sistema deixe de depender de rodada manual recorrente.
- Garantir que backend, frontend, rotas, integracoes e fluxos criticos possam ser validados de forma reproduzivel.
- Transformar a rotina de validacao em comando unico ou poucos comandos previsiveis.

**Escopo esperado:**
- Testes unitarios no backend para use cases, handlers, fila, processamento e regras de dominio.
- Testes unitarios no frontend para hooks, funcoes de mapeamento de estado, validacoes, componentes criticos e adaptadores de API.
- Testes de integracao entre rotas, services, storage, fila e banco nos fluxos mais sensiveis.
- Testes end-to-end cobrindo os fluxos principais reais do produto: autenticacao, upload, indexacao, monitoramento documental, preview/download, retry, exclusao, configuracao de provider e chat.
- Ambiente reproduzivel com devcontainers para subir dependencias, rodar a aplicacao e executar a bateria de testes sem setup manual repetitivo.
- Scripts claros para execucao local e em pipeline, com separacao minima entre `unit`, `integration`, `e2e` e `smoke`.
- Documentacao curta ensinando como subir o ambiente de testes e como interpretar falhas.
- Suporte a variaveis sensiveis em `.env` ou `.env.test` para cenarios que exigem credenciais reais de providers e bridges externas.
- Definir claramente quais segredos serao necessarios para os testes reais de integracao, por exemplo tokens e chaves de Ollama, OpenAI-compatible, storage e outros providers usados na validacao.
- Garantir que esses tokens nao sejam revelados, commitados, logados ou exibidos nos relatórios de teste, devcontainers e pipelines.
- Prever mocks e modo offline quando os segredos nao estiverem presentes, mas permitir rodada real quando as credenciais existirem no ambiente.

**Lista inicial de envs esperadas para a frente de testes:**
- Base da aplicacao e ambiente de teste:
  `DATABASE_URL`, `RABBITMQ_URL`, `JWT_SECRET`, `MASTER_ENCRYPTION_KEY`, `API_INTERNAL_URL`, `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_MAX_UPLOAD_SIZE_BYTES`.
- Storage para cenarios reais de integracao:
  `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET_NAME`, `MINIO_USE_SSL`.
- Providers LLM e bridges compativeis:
  `OPENAI_API_KEY`, `OPENAI_BASE_URL`, `OPENAI_MODEL`, `OLLAMA_API_KEY`, `OLLAMA_BASE_URL`, `OLLAMA_MODEL`, alem de envs equivalentes para outros providers realmente exercitados.
- Regras de seguranca:
  tokens, chaves e segredos devem existir apenas em `.env`, `.env.test`, secrets do devcontainer ou secrets da pipeline, nunca em arquivo versionado.
- Comportamento esperado:
  quando os segredos nao existirem, a suite deve conseguir rodar em modo mockado/offline; quando existirem, deve habilitar rodadas reais de integracao e healthcheck.

**Por que importa:**
- Evita regressao silenciosa em rotas, services, frontend e integracoes.
- Reduz drasticamente a necessidade de testar a aplicacao manualmente a cada mudanca.
- Aumenta confiabilidade para evolucao rapida do projeto, defesa tecnica do TCC e distribuicao open source.
- Cria evidencias objetivas de que o sistema continua funcionando como esperado depois de cada rodada de implementacao.

**Observacao:**
- Ja existem testes unitarios relevantes no frontend e backend, smoke HTTP no backend, smoke `e2e` inicial no frontend, devcontainer, uma primeira camada de integracao real no backend com PostgreSQL, MinIO e RabbitMQ, e uma primeira bateria `e2e` autenticada do backend.
- O backend agora possui testes de integracao do repositório documental, da outbox, do storage MinIO e da fila RabbitMQ com `go test -tags=integration ./...`.
- O backend agora tambem tem testes opcionais de provider LLM real e de processamento documental com IA via `go test -tags=integration ./internal/pkg/provider ./internal/processing`, habilitados por env e com `skip` por padrao quando as chaves nao estao presentes.
- O repositorio agora tambem expõe entrypoint unico de testes na raiz com `./scripts/test.sh` e atalhos via `Makefile`, reduzindo a necessidade de decorar comandos longos por stack.
- O backend agora cobre tambem casos negativos mais agressivos de upload e processamento, incluindo spoofing de MIME, falha de leitura do stream, erro inesperado na resolucao de storage, fallback do storage customizado para o base, arquivo realmente orfao, falha do parser e falha de persistencia do indice.
- No backend, a bateria autenticada ponta a ponta agora ja cobre tambem provider LLM, ativacao para chat, ciclo de conversa e stream de resposta grounded com provider fake OpenAI-compatible.
- No backend, a bateria `e2e` agora tambem cobre upload rejeitado por spoofing de MIME e payload excedido.
- O `e2e` do frontend agora cobre landing, documentacao, CTA para registro, redirect de rota protegida e redirect de auth para dashboard quando existe cookie de sessao.
- O frontend foi migrado de `middleware` para `proxy`, eliminando o warning de convencao do Next.js nessa frente.
- Os testes reais de integracao continuam podendo exigir segredos locais configurados no ambiente para validar envio, conexao e healthcheck de providers.
- O comando `pnpm run test:e2e:install` segue opcional para ambientes sem Chromium do sistema.
- O escopo original desta task foi fechado; ampliacoes futuras de cenarios entram como manutencao incremental, nao como lacuna estrutural.

### [A1] Fluxo de exclusao realmente confiavel de documentos

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Exclusao individual de documento.
- Exclusao em lote dos documentos de uma collection.
- Limpeza de dados derivados de indexacao.
- Atualizacao da UI apos exclusao.
- Confirmacao forte na UI para exclusao individual e em lote.
- Falha explicita quando a remocao do arquivo original no storage nao acontece.
- Auditoria minima persistida para sucesso e falha de exclusao.
- Auditoria de exclusao com actor, origem e correlacao por request.
- Cobertura de testes para cenarios de falha no storage durante exclusao.

**Observacao:**
- Os refinamentos restantes deixaram de ser parte de `A1`.
- Visualizacao operacional da auditoria fica melhor classificada em `B2`.
- Testes de integracao com storage real ja foram absorvidos na bateria automatizada concluida em `A0`.

**Por que importa:**
- O fluxo agora deixa de reportar sucesso silencioso quando a remocao fisica do arquivo falha.
- Isso reduz inconsistencias entre banco, storage e dashboard.

**Escopo entregue:**
- Excluir arquivo do storage com retorno consistente em caso de falha.
- Excluir ou invalidar dados derivados.
- Atualizar dashboard/listas.
- Manter confirmacao forte na UI.
- Registrar evento minimo de auditoria.

### [A2] Download explicito do arquivo original

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Endpoint de leitura do conteudo original.
- Rota/proxy dedicada para download do binario original.
- Acao clara de "baixar original" na interface.
- Separacao semantica entre preview inline e download.
- Mensagem visual mais clara quando o usuario esta inspecionando preview versus baixando o arquivo.

**Por que importa:**
- Fecha o ciclo operacional do catalogo documental.
- Melhora auditoria, validacao e debug.

**Observacao:**
- Refinos adicionais de UX foram absorvidos em `D6`.
- Eventuais ajustes extras de headers e hardening continuam melhor classificados em `C1`.

### [A3] Documento orfao com UX operacional melhor

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Deteccao do arquivo ausente em leitura/processamento.
- Marcacao do documento como `ORPHANED`.
- Retorno explicito quando o original nao e encontrado no storage.
- Destaque visual do estado orfao na UI documental.
- Orientacao operacional com acoes claras de retry, exclusao e reupload.
- Separacao visual melhor entre preview indisponivel, falha de indexacao e arquivo inexistente.

**Por que importa:**
- O estado orfao deixou de ser apenas tecnico e passou a virar fluxo operacional claro.

**Observacao:**
- Refinos adicionais de apresentacao foram absorvidos em `D6`.

### [A4] Padronizacao das confirmacoes destrutivas restantes

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Confirmacao forte em exclusao de chat.
- Confirmacao forte para excluir documento.
- Confirmacao forte para excluir documentos da collection.
- Componente reutilizavel para confirmacoes destrutivas com linguagem e severidade mais consistentes.
- Textos dos modais destrutivos centralizados em i18n.
- Semantica de dialogo, labels acessiveis e fechamento por `Escape` no modal reutilizavel.

**Observacao:**
- A task deixa de ser "espalhar mais modais" e passa a ser apenas manutencao normal caso apareca alguma nova acao destrutiva no futuro.

## 2. Prioridade Alta - Confiabilidade Operacional

### [B1] Fila, retry automatico e DLQ mais operacionais

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Pipeline assincrono funcional.
- Retry automatico na fila.
- DLQ configurada.
- Retry manual por documento na UI.
- Persistencia estruturada das tentativas operacionais de processamento.
- Classificacao explicita entre falha transitoria, falha permanente e original ausente.
- Exposicao do historico recente de tentativas no payload do documento.
- Painel operacional no frontend para inspecionar documentos em retry, DLQ e falha permanente.
- Cobertura automatizada de backend e frontend para a nova leitura operacional.

**Por que importa:**
- A infraestrutura deixou de ser apenas funcional e passou a ter leitura operacional real.
- Este item deixa de ser "ter retry e DLQ" e passa a virar "entender por que falhou, em que tentativa, com qual classificacao e qual acao cabe agora".

**Observacao:**
- Refinos futuros de telemetria e logs ainda podem evoluir em `D2`, mas a lacuna estrutural de observabilidade operacional de retry/DLQ foi fechada neste escopo.

### [B2] Trilhas de auditoria basicas

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Upload.
- Reindexacao / retry.
- Exclusao.
- Alteracoes sensiveis de provider.
- Alteracao de ativacao de provider para chat/indexacao/storage.
- Explorers dedicados para auditoria documental e de providers.
- Resumo agregado com totais, requests, cobertura e ultimo evento.
- Filtros operacionais por busca, acao, status, actor, actor id, request id e janela de tempo.
- Recorte contextual por documento, collection, familia e provider ativo.
- Atalhos rapidos para falhas e acoes mais frequentes.
- Paginacao incremental e cobertura automatizada em unit e `e2e`.

**Por que importa:**
- Aumenta rastreabilidade.
- Ajuda manutencao futura.
- Fortalece defesa tecnica do TCC e postura de produto serio.

**Observacao:**
- A trilha deixou de ser apenas minima.
- O foco agora ja pode sair de auditoria basica e ir para portabilidade e manutencao incremental, sem lacuna estrutural nessa frente.

### [B3] Healthcheck de provider mais completo e mais legivel

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Teste de conexao antes de salvar.
- Validacao de credenciais em varios cenarios.
- Classificacao basica de falha por tipo: credencial, autenticacao, endpoint, timeout e modelo invalido.
- Respostas HTTP e mensagens mais legiveis para os cenarios principais de healthcheck.
- Cobertura de testes para timeout, falha de rede, autenticacao e modelo invalido.
- Relatorio estruturado de readiness no backend com checks explicitos por credenciais, conectividade e discovery.
- Painel visual de readiness no frontend para settings.
- Discovery de modelos para Ollama e providers OpenAI-compatible suportados no runtime atual.
- Cobertura automatizada adicional em backend, frontend e `e2e`, alem de docs de API atualizadas no Swagger/Scalar.

**Por que importa:**
- Reduz configuracao quebrada silenciosa.
- Melhora UX e DX operacional.

### [B4] Correcao e rastreabilidade do bug de upload percebido

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Diagnostico fechado:**
- Nao apareceu evidencia de sucesso falso no backend de upload.
- Os cenarios sensiveis do runtime ja estavam sendo rejeitados corretamente pela API, especialmente MIME falsificado, extensao enganosa e payload excedido.
- O problema concreto era rastreabilidade ruim no frontend: rejeicoes reais da API e falhas do proxy podiam parecer apenas um erro generico de upload.

**Evidencia minima registrada:**
- O frontend agora gera e envia `X-Request-ID` em chamadas autenticadas.
- A rota proxy de upload preserva esse `request id` ate o backend e o devolve na resposta.
- As mensagens de erro do upload passaram a incluir `Request ID`, o que diferencia rejeicao real da API de falha de proxy sem depender de inspeção manual.
- A cobertura automatizada do frontend agora valida essa rastreabilidade em testes unitarios do cliente de upload e do hook de upload.

**Observacao:**
- Se um problema de upload voltar a ser percebido, agora existe um identificador direto para cruzar a tentativa com logs e auditoria, sem manter esta task aberta como suspeita generica.

### [B5] Cobertura de testes para fluxos criticos

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Observacao importante:**
- O conjunto de lacunas que motivava `B5` foi fechado no escopo atual.
- Novos cenarios de regressao continuam desejaveis, mas agora entram como manutencao incremental normal da suite.

**Ja existe:**
- Cobertura de exclusao com falha no storage.
- Cobertura do healthcheck de provider com falhas de rede, timeout, credenciais invalidas e modelo invalido.
- Cobertura dos fluxos minimos de auditoria adicionados recentemente em handlers e use cases.
- Testes de regressao para a correcao do indexador deterministico de PDF.
- Testes de integracao reais com PostgreSQL para repositório documental e outbox.
- Testes de integracao reais com MinIO para upload, leitura, existencia e remocao do arquivo.
- Testes de integracao reais com RabbitMQ para publish/consume, retry e DLQ.
- Testes opcionais com provider LLM real para validacao de conexao, geracao de resposta e processamento documental enriquecido com IA quando as credenciais estao presentes no ambiente.
- Testes `e2e` do backend cobrindo autenticacao, upload, listagem, leitura do original, retry, documento orfao, exclusao, configuracao de provider LLM, ativacao para chat, ciclo de conversa e stream grounded.
- Testes `e2e` do backend cobrindo tambem rejeicao de upload por MIME falsificado e por payload excedido no fluxo real da API.
- Testes `e2e` do frontend cobrindo smoke publico com landing, navegacao para documentacao e redirect de rota protegida, com fallback para o Chromium do sistema.
- Testes `e2e` do frontend cobrindo tambem CTA para registro e redirect de rota de auth para dashboard quando ja existe sessao.
- Testes unitarios negativos de upload contra spoofing de MIME, extensao incoerente e falha de leitura do stream.
- Testes do `DocumentProcessor` para falha de resolucao de storage, fallback de leitura, documento orfao, erro de parser e falha ao persistir o indice.

**Por que importa:**
- A frente de testes deixou de ser uma lacuna estrutural e passou a ter base real para evolucao incremental.

## 3. Prioridade Alta - Seguranca e Robustez Open Source

### [C0] Integridade automatica antes de commit

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- `pre-commit` versionado em `.husky/pre-commit`.
- Script central em `scripts/pre-commit-checks.sh`.
- Entry point de testes na raiz em `scripts/test.sh`.
- Atalhos adicionais em `Makefile` para `test-unit`, `test-backend-integration`, `test-backend-e2e`, `test-frontend-e2e` e `test-all`.
- Validacao seletiva do backend com `gofmt`, `go vet` e `go test ./...` quando houver mudancas em `api/`.
- Validacao seletiva do frontend com `pnpm lint`, `pnpm run typecheck` e `pnpm run test:unit` quando houver mudancas relevantes em `frontend/`.

**Observacao:**
- O hook foi mantido propositalmente fora de `integration` e `e2e` para nao tornar o commit local inviavel.
- A bateria mais pesada continua documentada em `A0`.

### [C1] Revisao de seguranca orientada a abuso e configuracao

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Criptografia de credenciais.
- Validacoes de endpoint.
- Mitigacoes de SSRF.
- Validacoes de upload e respostas sanitizadas.
- Revisao inicial de headers e comportamento das rotas de preview/download.
- Checklist versionada em `SECURITY.md`.
- Sanitizacao de `Content-Disposition` no backend e no proxy do frontend.
- Bloqueio explicito de preview inline para MIME nao seguros, inclusive HTML armazenado.
- Headers adicionais de hardening em preview/download (`no-store`, `nosniff`, `no-referrer` e protecoes de embedding no proxy).
- Cobertura automatizada objetiva para upload abuse, restricao de preview, sanitizacao de filename e validacao de endpoints/provider URLs.

**Por que importa:**
- Agrega valor real para um projeto open source.
- Fortalece o discurso de engenharia responsavel no TCC.

### [C2] Remover dados estaticos enganosos ou desatualizados do frontend

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Dashboard sem telemetria estatica enganosa de uso mensal, expiracao fixa de sessao ou custo predefinido.
- Card de custo reescrito para deixar explicito que precificacao real depende de precos sincronizados e uso observado.
- Bloco visual de qualidade reescrito para pedir validacao por fontes, sem score fixo artificial.
- Cobertura unitária no frontend para impedir o retorno de placeholders como `1.2k / 5k tokens`, `Expires in 4h`, `Vertex AI / Gemini 1.5` e valores fixos de custo.

**Por que importa:**
- Reduz ambiguidade para usuarios e avaliadores.
- Melhora honestidade tecnica do projeto.

**Observacao:**
- Fallbacks de modelos e catálogos estruturais continuam podendo existir quando sao claramente apresentados como configuracao ou sugestao, e nao como telemetria real.
- Refinos futuros dessa natureza deixam de ser uma task alta isolada e passam a entrar como manutencao normal de copy e UX.

### [C3] Internacionalizacao das partes que ainda estao faltando

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- i18n amplo em backend e frontend.
- Mensagens novas de healthcheck de provider localizadas no backend.
- Catalogo de mensagens ampliado para cobrir auditoria, documentos, settings e fallbacks operacionais.
- Explorers de auditoria revisados para `pt-BR`, `en` e `es`, incluindo filtros, atalhos, resumo e cards de evento.
- Pagina de documentos revisada para remover mistura residual de PT/EN em monitoramento, preview, fila operacional, toasts e banners de falha.
- Fallbacks de `lib/api/*` e do preview inline do Next passaram a respeitar locale em vez de respostas fixas em portugues.
- Suite automatizada atualizada para validar textos localizados em `unit` e `e2e`.

**Por que importa:**
- Aumenta polimento do produto.
- Mostra consistencia arquitetural.

## 4. Prioridade Media - Qualidade do Produto de IA

### [D1] Acuracia de indexacao com avaliacao real

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Score heuristico.
- Telemetria e monitoramento.
- Testes relacionados a recuperacao e corpus de avaliacao.
- Benchmark publico em `GET /api/v1/meta/indexing-benchmark`.
- Corpus controlado versionado com ground truth explicito em `api/internal/usecase/testdata/indexing-benchmark/corpus.json`.
- Metricas objetivas para estrutura, retrieval documental, retrieval por collection e exact match.
- Narrativa publica sobre metodologia, criterios de passagem, limitacoes e leitura correta do score heuristico.
- Runner dedicado via `./scripts/indexing-benchmark.sh` e comandos associados para reproduzir a baseline atual.

**Por que importa:**
- Este eh um dos itens que mais fortalecem o valor academico e tecnico do projeto.

**Observacao:**
- A task era fechar medicao objetiva e reproduzivel, nao prometer benchmark perfeito ou generalizacao irrestrita para qualquer corpus.
- A partir daqui, ampliacoes de corpus, comparacoes contra outras estrategias de retrieval e refinamentos de citacao grounded entram melhor como evolucao incremental e pesquisa futura, nao como lacuna estrutural do runtime atual.

### [D2] Observabilidade operacional mais madura

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Pagina e resumo de monitoramento.
- Dados de custo, tokens, requests, score estimado e modelo.
- Slice operacional na tela de documentos com busca, watchlist prioritaria e filtros acionaveis por atencao, intervencao manual, travamento, retry e falhas recentes.
- Heuristica compartilhada e testada para distinguir estados nominais, travados, retry agendado, DLQ, falha permanente e orfandade.
- Biblioteca que troca a navegacao hierarquica por lista direta quando o operador aplica um recorte operacional.
- Resumo e documentacao da API alinhados para deixar claro que `GET /documents/monitoring/summary` alimenta o painel operacional do frontend.
- Cobertura automatizada em unit, rota `app/api` e `e2e` para impedir regressao dessa leitura operacional.

**Por que importa:**
- Transforma telemetria em ferramenta operacional real.

### [D3] Feedback mais claro na UI sobre configuracao ativa de LLM

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Configuracao separada de providers por papel.
- Monitoramento mostra provider/modelo em varios pontos.
- Endpoint dedicado de runtime ativo para chat e indexacao.
- Callouts reutilizaveis no fluxo principal de chat e documentos mostrando provider, modelo e ultimo update.
- CTA direto para configuracoes quando o runtime ativo precisa ser ajustado.
- Cobertura automatizada em backend, frontend e smoke visual para impedir regressao dessa leitura.

**Observacao:**
- O gap deixou de ser "descobrir qual LLM esta em uso" e passou a ser manutencao incremental da UX em torno do runtime ativo.
- Refinos adicionais de copy, layout ou onboarding entram melhor em manutencao normal de UX do chat.

### [D4] Melhorar a legibilidade e a experiencia do chat

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Streaming SSE.
- Referencias de fonte.
- Progresso de subetapas.
- Historico e exclusao de conversas.
- Workspace dedicado com hierarquia visual melhor, hero contextual e rail interna de conversas.
- Estados vazios e feedbacks mais claros para loading, erro, runtime ausente e novo rascunho.
- Composer refinado com atalhos de prompt, foco mais claro e leitura operacional da collection ativa.
- Leitura das respostas lapidada com rich text basico, badges de actor, timestamp e fontes expansiveis.
- Logica do chat extraida para hook dedicado e componentes menores, reduzindo acoplamento do `ChatTab`.
- Cobertura automatizada nova em unit e `e2e` para o workspace de chat.
- Contrato OpenAPI versionado agora documenta as rotas de chat usadas pelo frontend em `Swagger/Scalar`.

**Observacao:**
- O gap deixou de ser estrutural.
- Refinos futuros aqui entram como manutencao incremental de copy e polimento visual, nao como task aberta principal.

### [D5] Melhorar a pagina de custo e a leitura de pricing

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Card de custo reestruturado como painel executivo, sem placeholders fixos ou telemetria artificial.
- Leitura explicita de confiabilidade entre `setup`, `cobertura parcial` e `estimativa confiavel`.
- Cruzamento entre runtime ativo, fontes oficiais de pricing sincronizadas e telemetria documental observada antes de exibir leitura de custo.
- Comunicacao clara sobre origem do numero, o que entra no calculo e o que fica de fora da estimativa.
- Lista de saude das fontes de pricing com status, ultima sincronizacao, quantidade de precos e erro mais recente.
- Drivers de custo observados por modelo, vindos da telemetria real de indexacao.
- Cobertura automatizada em unit e `e2e`, alem de documentacao do fluxo no README do frontend e na pagina `/documentation`.
- Contrato OpenAPI/Scalar do backend anotado para as rotas de pricing consumidas pelo frontend.

**Por que importa:**
- A tela deixou de sugerir um numero solto e passou a comunicar maturidade, cobertura e limite operacional da estimativa.
- Isso reduz percepcao enganosa de custo e fortalece a honestidade tecnica do produto.

**Observacao:**
- O que sobra aqui vira manutencao incremental de copy, UX e evolucao futura do runtime de pricing, nao backlog estrutural.

### [D6] Melhorar a pagina de monitoramento documental

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Telemetria ampla por documento e resumo agregado.
- Leitura executiva consolidada no topo do painel.
- Proximas acoes operacionais transformadas em recortes acionaveis.
- Mapa de anomalias por tipo com hints e foco rapido.
- Watchlist prioritaria com orientacao operacional mais explicita.
- Cobertura automatizada em unit, Storybook interaction e `e2e`.

**Por que importa:**
- A pagina deixou de ser apenas um conjunto de cards e passou a orientar decisao operacional.
- O operador agora enxerga prioridade, anomalia e proxima acao sem precisar interpretar toda a biblioteca manualmente.

## 5. Prioridade Media - Portabilidade e Ecossistema

### [E1] Implementar de fato os outros storages

**Status:** NAO IMPLEMENTADO no runtime principal

**Ja existe:**
- Arquitetura preparada para provider de storage.
- MinIO / S3-compatible consolidado.

**Falta:**
- Implementacoes concretas adicionais realmente exercitadas no runtime.

**Observacao:**
- Importante para produto open source, mas nao vem antes de fechar confiabilidade e fluxos principais.

### [E2] Revisao real do suporte aos outros providers de LLM

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Suporte estrutural a varios providers.
- Teste de conexao.
- Configuracao por papel.
- Catalogo de modelos hardcoded no frontend como decisao atual.
- Readiness mais consultavel no runtime atual, com retorno estruturado e melhor discovery para providers OpenAI-compatible e Ollama.
- Matriz publica de suporte real por provider LLM, com classificacao entre `READY`, `PARTIAL` e `EXPERIMENTAL`.
- Exposicao dessa matriz na API publica, no Swagger/Scalar, na pagina `/documentation` e na tela de settings.
- Defaults e onboarding revisados para nao sugerirem maturidade maior do que a real, inclusive com notas operacionais e requisitos de bridge.
- Exemplos HTTP, devcontainer, doctor e docs principais apontando para o endpoint publico de suporte.
- Cobertura automatizada em backend e frontend para rota, proxy, UI e leitura documental da matriz.

**Observacao:**
- `D2.1` nao deve mais aparecer como faltando.
- Por decisao atual, os modelos podem continuar hardcoded no frontend.
- O gap deixou de ser "ter suporte estrutural sem classificacao honesta" e virou manutencao incremental da matriz conforme o runtime evoluir.

### [E3] Tornar a API e o projeto mais faceis de consumir como open source

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- README raiz, README do backend, README do frontend e guia de onboarding alinhados ao runtime atual.
- Trilha explicita para `self-hosted` e para consumo de `hosted API`, sem misturar os dois cenarios.
- Comandos previsiveis de `bootstrap`, `stack` e `doctor` para setup, subida e troubleshooting.
- Devcontainer com fluxo reproduzivel para subir o ambiente e validar a stack.
- Pagina de documentacao com onboarding separado por contexto de uso.
- Referencia de API com `Scalar` como interface principal, `Swagger UI` como fallback e `OpenAPI JSON` como contrato consumivel.
- Exemplo HTTP reaproveitavel trocando apenas a base da API.
- Cobertura automatizada em backend e frontend para impedir regressao nos caminhos de documentacao e onboarding.

**Observacao:**
- O gap deixou de ser estrutural.
- O que sobra aqui vira manutencao incremental de docs, exemplos e copy, nao backlog principal.

### [E4] Melhorar onboarding para agentes e colaboradores automatizados

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Setup, docs e fluxo de subida mais claros para humanos e agentes que operam no repositorio.
- Menos ambiguidade de comandos com wrappers explicitos para bootstrap, stack e diagnostico.
- Ambiente reproduzivel via devcontainer para contribuicao local com menos setup manual.
- Separacao mais clara entre runtime local, API hospedada, docs de referencia e exemplos de consumo.
- Diagnostico automatizado capaz de validar stack local ou deployment remoto com `doctor --api-base`.
- Contrato de API e documentacao navegavel expostos no proprio runtime, reduzindo dependencia de conhecimento tacito do repositorio.

**Observacao:**
- O que resta nessa frente eh manutencao normal de contribuicao e documentacao.
- Nao faz mais sentido manter esta task como pendencia estrutural.

### [E5] CLI para importar indices baixados para banco de dados

**Status:** NAO IMPLEMENTADO

**Objetivo:**
- Permitir que qualquer pessoa aponte uma conexao de banco e um path local com os arquivos baixados da nossa pagina.
- Criar automaticamente as tabelas necessarias para armazenamento dos indices.
- Inserir ou sincronizar os dados importados no banco sem exigir setup manual de schema.

**Escopo esperado:**
- CLI com parametros para `database_url` e `path` do arquivo ou diretorio exportado.
- Validacao do formato esperado dos arquivos de indice.
- Criacao automatica de tabelas e indices necessarios no banco alvo.
- Importacao idempotente ou com estrategia clara de conflito.
- Logs minimos de execucao com quantidade de registros importados, ignorados e falhos.
- Documentacao curta com exemplo de uso e pre-requisitos.

**Por que importa:**
- Facilita distribuicao do valor do projeto para alem da interface web.
- Reduz friccao para quem quiser reutilizar os dados indexados em outro ambiente.
- Aumenta o valor open source ao transformar o pipeline em algo reaproveitavel por CLI.

## 6. Prioridade Media - Documentacao e Consistencia do Repositorio

### [F1] Atualizar README, READMEs internos e pagina de documentacao para o runtime atual

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- README principal.
- README do backend.
- README do frontend.
- Pagina de documentacao.
- Indice novo para docs secundarias em `docs/README.md`.
- `docs/functional-readiness-spec.md` reescrito para refletir o runtime atual.
- `docs/proposta-continuacao-tcc.md` reescrito como proposta de pesquisa derivada do ThingIndex atual.
- Pagina `/documentation` agora separa runtime atual, prontidao atual e pesquisa/roadmap.
- Roadmaps e tasklists em `docs/roadmap/` agora carregam aviso explicito de que sao trilhas de pesquisa, nao descricao da implementacao principal.

**Por que importa:**
- Agrega valor real para open source.
- Melhora onboarding e confiabilidade da apresentacao do TCC.

### [F2] Explicitar melhor o que eh prototipo, o que eh funcional e o que eh roadmap

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- README raiz e READMEs principais agora separam melhor funcionalidade atual, limitacoes do runtime e roadmap.
- O backlog principal ja deixa mais claro o que esta concluido, o que esta parcial e o que e expansao.
- Docs secundarias agora classificam explicitamente o que e runtime atual, o que e prontidao atual e o que e trilha academica.
- Arquivos antigos que ainda citam GraphRAG, Neo4j, ChromaDB ou stacks alternativos passaram a carregar contexto de pesquisa e roadmap.
- A pagina de documentacao do frontend agora orienta a leitura correta do repositorio por camada documental.

**Por que importa:**
- Aumenta honestidade tecnica.
- Melhora defesa academica.

### [F3] Revisar boilerplate, naming e componentes genericos demais

**Status:** IDEIA CONTINUA

**Reescrita melhor:**
- Revisar partes repetitivas ou com cara de scaffolding.
- Simplificar naming, comentarios e componentes desnecessarios.
- Manter apenas abstracoes que agregam manutencao real.

### [F4] Instalar e estruturar Storybook para os componentes do frontend

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- Storybook instalado e integrado ao `frontend` com framework `@storybook/nextjs-vite`.
- Scripts previsiveis para desenvolvimento, build e smoke `e2e` do catalogo visual.
- Stories iniciais cobrindo `components/ui`, runtime de provider, readiness de settings e observabilidade documental.
- Decorator compartilhado para i18n e layout base, com seletor de idioma no toolbar.
- Pagina introdutoria em MDX com convencoes e fluxo de validacao.
- Smoke unitario para stories criticas e smoke `e2e` com Playwright sobre docs e componentes centrais.
- Porta `6006` encaminhada no devcontainer e comandos expostos na raiz via `scripts/test.sh` e `Makefile`.
- Documentacao atualizada para uso local e dentro do container.

**Observacao:**
- Storybook nao substitui testes unitarios, integracao ou e2e.
- O maior valor inicial foi fechado cobrindo a base de `ui` e os componentes operacionais mais recorrentes; a expansao futura de stories vira manutencao incremental.

**Por que importa:**
- Aumenta clareza do frontend para manutencao e contribuicao open source.
- Ajuda a detectar regressao visual e inconsistencia de estados antes de chegar no fluxo real.
- Fortalece a apresentacao tecnica do projeto ao mostrar organizacao da camada de interface.

## 7. Prioridade Baixa - Valor Academico e Posicionamento

### [G1] Criar material academico derivado do projeto

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- rascunho versionado do artigo em `article/thingindex-sbc-article.md`;
- workspace academico em `article/README.md`;
- referencia direta ao template SBC em `article/sbc_template (1).doc`;
- articulacao com foco do produto em `rfc/0001-product-focus.md`;
- endpoint publico de perfil do runtime para sustentar narrativa tecnica e academica.

**Por que importa:**
- Transforma implementacao em contribuicao defendivel no TCC.

### [G2] Validar proposta de valor e foco do projeto

**Status:** CONCLUIDO NO ESCOPO ATUAL

**Ja existe:**
- caso de uso principal formalizado em `rfc/0001-product-focus.md`;
- landing e `/documentation` alinhadas ao foco atual do produto;
- perfil publico do runtime em `GET /api/v1/meta/runtime-profile`;
- README e docs operacionais atualizados para separar runtime atual de pesquisa futura.

### [G3] Refinos visuais menores

**Status:** BAIXA PRIORIDADE

**Exemplos:**
- Icones de IAs.
- Pequenos polimentos cosmeticos.

## 8. Itens Que Ja Nao Devem Ficar Como "Faltando"

Estes itens ja existem ao menos em nivel funcional e nao devem voltar como "nao implementado":

- Chat com endpoint real e resposta incremental por streaming.
- Exclusao individual de documento.
- Exclusao em lote por collection.
- Fluxo de exclusao confiavel com falha explicita de storage.
- Auditoria minima de exclusao.
- Auditoria de exclusao com actor, origem e correlacao.
- Auditoria minima de upload, retry/reindexacao e alteracoes sensiveis de provider.
- Padrao reutilizavel de confirmacao destrutiva para os fluxos principais de exclusao.
- Download explicito do arquivo original com separacao clara entre preview e download.
- Deteccao e marcacao basica de documento orfao.
- Exposicao operacional basica do estado orfao na UI documental.
- Teste de conexao de provider antes de salvar.
- Healthcheck de provider com distincao basica entre credencial, autenticacao, endpoint, timeout e modelo.
- Retry automatico na fila.
- DLQ basica.
- Botao de retry / reindexacao de documento.
- Bateria automatizada com `unit`, `integration`, `e2e` e `smoke` documentada por comandos unificados.
- Suporte a varios providers de LLM no catalogo estrutural do projeto.
- Suporte a Ollama com token opcional em cenarios compativeis.
- Pagina de monitoramento documental com telemetria relevante.
- Configuracao separada de LLM para indexacao e chat.
- Listagem de modelos como capacidade de backend.
- Benchmark publico de indexacao com corpus controlado, metricas objetivas e interpretacao explicita do score heuristico.

**Observacao:**
- Alguns desses ainda precisam de refinamento.
- Em especial, listagem de modelos nao precisa entrar no backlog agora porque a decisao atual eh manter modelos hardcoded no frontend.

## 9. Ordem Sugerida de Implementacao

### Fase 1 - Consolidar qualidade percebida do produto

- Nenhuma task aberta restante nesta fase.

### Fase 2 - Expandir portabilidade e simplificar manutencao

- `E1` Implementar novos storages.
- `E5` Implementar CLI para importar indices em banco externo.
- `F3` Revisar boilerplate e naming desnecessarios.

### Fase 3 - Roadmap de expansao e valor academico

- `G3` Aplicar polimentos visuais menores.

## 10. Resumo Curto

Se a pergunta for "quantas tasks existem e quantas sao realmente urgentes":

- Existem 32 tasks catalogadas.
- 28 ja podem ser tratadas como concluidas no escopo atual.
- Das 4 abertas, nenhuma permanece como alta necessidade estrutural agora.

Se a pergunta for "todas sao ultra necessarias":

- Nao.
- O backlog tem uma mistura de risco real, melhoria incremental, DX, open source, roadmap e valor academico.
- O erro seria tratar tudo como igualmente urgente.

Se a ideia for escolher o proximo passo mais util agora:

- `E1` implementar storages adicionais para reduzir dependencia pratica do stack S3-compatible.
- `E5` criar a CLI para importar indices em banco externo.
- `F3` reduzir boilerplate e naming excessivo enquanto o backlog estrutural esta curto.

Se a ideia for enxergar melhor o backlog:

- Use a tabela-resumo deste arquivo para decidir por `Necessidade` e `Tamanho`.
- Trate `Fase 1` como fila curta de execucao.
- Trate `Fase 3` como backlog de expansao, nao como bloqueio do produto.
