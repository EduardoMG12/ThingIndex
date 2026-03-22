# Frontend

O frontend do ThingIndex e uma aplicacao Next.js 16 responsavel por:

- autenticacao do usuario;
- configuracao de providers de storage e LLM;
- upload e navegacao de documentos;
- visualizacao de colecoes, categorias e contexto;
- chat grounded por collection com historico persistido e resposta incremental;
- auditoria consultavel de documentos e providers, com filtros, resumo agregado e paginação incremental.

O app atua como camada de experiencia sobre a API Go do monorepo. Ele nao implementa logica de indexacao no browser; a maior parte das operacoes de negocio e delegada para o backend por rotas `app/api/*` que fazem proxy autenticado.

## Stack

- Next.js 16
- React 19
- TypeScript
- Tailwind CSS
- Framer Motion

## Estrutura relevante

```text
src/
|-- app/                 # rotas, layouts e API routes do Next.js
|-- components/          # interface reutilizavel
|-- hooks/               # hooks de consulta e estado
|-- lib/api/             # clientes e contratos de API
|-- lib/i18n/            # mensagens e infraestrutura de localizacao
```

## Principais telas

- landing page institucional;
- login e registro;
- dashboard com abas de upload, documentos, chat e custo;
- tela de configuracao de providers;
- documentacao visual da arquitetura.

## O que ja funciona

- login, registro e persistencia de sessao por cookie `httpOnly`;
- proxy autenticado para rotas protegidas da API;
- upload de documentos;
- listagem de categorias, collections e documentos;
- painel lateral de preview e auditoria da indexacao;
- painel operacional de documentos com destaque para falhas recentes, estados travados, watchlist prioritaria e slice acionavel da biblioteca;
- explorer de auditoria documental com busca, actor, request id, recorte por documento/collection, atalhos de falha e carregamento incremental;
- explorer de auditoria de providers com escopo por familia ou provider ativo, filtros operacionais e atalhos para saves, ativacoes e falhas;
- exibicao de monitoramento por documento:
  - status, etapa e duracao;
  - tokens estimados;
  - custo estimado;
  - modelo ativo;
  - score heuristico de qualidade;
- resumo agregado de monitoramento por usuario na tela de documentos;
- configuracao de providers com separacao entre LLM de indexacao e LLM de chat;
- relatorio visual de readiness na tela de settings com checks de credenciais, conectividade e discovery de modelos;
- fluxo real de chat por collection com conversas persistidas, referencias de fonte e streaming incremental;
- internacionalizacao em portugues, ingles e espanhol.

## Limitacoes atuais

- o chat runtime depende de provider LLM habilitado para chat e ainda precisa de validacao objetiva de acuracia e citacao;
- o preview inline foi intencionalmente restringido para formatos seguros, evitando execucao de HTML enviado pelo usuario;
- a maior parte da telemetria vem do backend; o frontend hoje exibe esses dados, mas nao e a origem deles.

## Desenvolvimento local

Instale as dependencias:

```bash
pnpm install
```

Execute em modo desenvolvimento:

```bash
pnpm dev
```

Abra o catalogo visual local:

```bash
pnpm storybook
```

Esse comando gera o build do Storybook e publica o catalogo em `http://localhost:6006`.

Se voce estiver chegando ao repositório pela primeira vez, o fluxo recomendado continua sendo:

```bash
./scripts/bootstrap.sh
./scripts/stack.sh up
./scripts/doctor.sh
```

Depois disso, abra a interface em `http://localhost:3000` e a documentação da API em `http://localhost:8000/docs`.

Materiais de apoio imediatos:

- guia curto: `docs/open-source-onboarding.md`
- requests prontas: `examples/http/thingindex-api.http`
- contrato versionado: `api/docs/swagger.yaml` e `api/docs/swagger.json`

Se o frontend estiver apontando para uma API já publicada, a documentação de onboarding e a página `/documentation` agora separam explicitamente o fluxo `self-hosted` do fluxo `hosted API`.

Valide o codigo:

```bash
pnpm lint
```

Quando houver mudancas relevantes no frontend, o `pre-commit` da raiz tambem executa automaticamente:

```bash
pnpm lint
pnpm run typecheck
pnpm run test:unit
```

## Testes

Comandos simples a partir da raiz do monorepo:

```bash
./scripts/test.sh frontend-unit
./scripts/test.sh frontend-e2e
./scripts/test.sh frontend-smoke
```

Ou, se preferir:

```bash
make test-frontend-unit
make test-frontend-e2e
make test-frontend-smoke
make test-frontend-storybook
```

Suite unitaria:

```bash
pnpm run test:unit
```

Smoke `e2e` com Playwright:

```bash
pnpm run test:e2e:install
pnpm run test:e2e
```

Somente os cenarios marcados como smoke:

```bash
pnpm run test:smoke
```

Storybook:

```bash
pnpm run build:storybook
pnpm run test:storybook:e2e
```

## Storybook

- a configuracao mora em `.storybook/` com `@storybook/nextjs-vite`, tema proprio e decorator global de i18n;
- `pnpm storybook` gera o catalogo e o expõe localmente na porta `6006`, enquanto `pnpm run build:storybook` valida apenas o build estatico;
- o toolbar permite alternar `en`, `pt-BR` e `es`, o que ajuda a revisar componentes que dependem de traducao;
- as stories iniciais cobrem primitives de `ui`, runtime/configuracao de providers e observabilidade documental;
- o smoke browser em `tests/storybook/storybook-smoke.spec.ts` garante que docs e stories centrais sobem e renderizam no build atual.

## Variaveis importantes

| Variavel | Exemplo | Descricao |
| --- | --- | --- |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8000/api/v1` | URL publica consumida pelo navegador |
| `API_INTERNAL_URL` | `http://api:8080/api/v1` | URL interna usada pelas rotas de proxy do Next.js |

## Integracao com a API

As rotas em `src/app/api/*` encapsulam:

- leitura do cookie de autenticacao;
- repasse de headers de idioma;
- limpeza de cookie quando a API responde `401`;
- isolamento do browser em relacao a detalhes internos da API.

Esse desenho reduz acoplamento entre componentes React e detalhes de transporte, e facilita futura evolucao para streaming, cache seletivo e politicas de sessao mais refinadas.

## Convencoes de i18n

- componentes React devem consumir `useI18n()` e evitar strings literais visiveis;
- modulos fora de React podem usar `src/lib/i18n/translate.ts` para localizar mensagens de runtime e fallbacks;
- rotas em `src/app/api/*` devem gerar mensagens localizadas na borda, sem empurrar copy fixa para o browser;
- mensagens operacionais de auditoria, documentos e settings devem nascer por chave, nao por string solta no componente.

## Observabilidade documental

- a tela `src/app/documents/page.tsx` combina o resumo agregado do backend com heuristicas locais de operacao para destacar documentos travados, retries, DLQ, orfaos e falhas recentes;
- a heuristica compartilhada mora em `src/lib/document-state.ts` e `src/lib/document-observability.ts`, evitando espalhar classificacao operacional pelo JSX;
- quando o usuario aplica busca ou slice operacional, a biblioteca troca a hierarquia por uma lista acionavel para inspecao direta dos documentos afetados.

## Observacao de produto

O frontend atual ja cobre o fluxo principal de autenticacao, upload, organizacao documental, auditoria da indexacao, Storybook para catalogo visual e um fluxo real de chat por collection com resposta incremental. Os proximos saltos mais uteis nesta camada estao em onboarding open source, refinamento de UX e limpeza de boilerplate residual.
