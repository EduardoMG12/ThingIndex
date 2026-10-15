<div align="center">
  <img src="assets/palmas-horizontal-1.png" alt="IFPR Campus Palmas" width="300" />

  <h1>ThingIndex</h1>

  <p><strong>Inteligência Documental com Proveniência Verificável sobre Modelo Relacional</strong></p>

  <p>Trabalho de Conclusão de Curso · Sistemas de Informação · IFPR Campus Palmas · 2026<br/>
  Charles Eduardo Mello Guimarães</p>

  <p>
    <a href="docs/article/ThingIndex-artigo.pdf"><strong>📄 Ler o artigo (PDF)</strong></a> ·
    <a href="#anexos"><strong>📎 Anexos</strong></a> ·
    <a href="#como-executar"><strong>▶️ Como executar</strong></a>
  </p>

  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Next.js-16-000000?logo=next.js&logoColor=white" alt="Next.js" />
  <img src="https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/RabbitMQ-fila-FF6600?logo=rabbitmq&logoColor=white" alt="RabbitMQ" />
  <img src="https://img.shields.io/badge/licen%C3%A7a-MIT-green" alt="Licença MIT" />
</div>

---

O ThingIndex responde perguntas sobre coleções de PDFs (leis, normas, manuais) e mostra
**de onde veio cada resposta**: o documento e a página, com o parágrafo destacado no PDF.

Cada parte do documento (página, parágrafo, linha de tabela, seção do sumário) é guardada
numa tabela do PostgreSQL, ligada ao documento por chave estrangeira e com a página de
origem. A busca é por palavras (BM25), sem banco vetorial. O LLM só entra no fim, para
redigir a resposta a partir das fontes que a busca escolheu. Por isso a página mostrada ao
usuário vem do banco, e não do texto escrito pelo modelo.

**Pergunta de pesquisa:** *é possível sustentar um sistema de inteligência documental com
auditabilidade utilizando exclusivamente modelagem relacional, sem depender de bancos
vetoriais ou grafos como camada primária de recuperação?*

<p align="center">
  <img src="docs/article/anexos/telas/workspace.png" alt="Workspace do ThingIndex" width="900" />
  <br/>
  <sub>Workspace de uma coleção de normas de segurança do trabalho: chat com as fontes, PDF na
  página citada com o parágrafo destacado e resumo do documento.</sub>
</p>

## Principais resultados

| Pergunta | Resultado |
|---|---|
| Busca por palavras ou por significado: qual acha mais? | Ficaram próximas: 13 de 16 contra 11 de 16 consultas, sem diferença detectável |
| Buscar trechos custa menos que mandar o PDF inteiro ao modelo? | Sim: na mediana, cerca de 1/26 dos *tokens*, com 72 de 72 perguntas certas contra 71 de 72 |
| Quanto o sistema acerta em perguntas novas? | 69,5% das respostas, cerca de 7 em 10 |
| A redação da pergunta muda o acerto? | Sim: 88,2% com as palavras do documento e 41,9% com outras palavras |
| Expandir a pergunta com um LLM ajuda? | De 69,5% para 88,7%; 14 perguntas ganhas e nenhuma perdida (experimental, desligada por padrão) |
| Fonte certa garante resposta certa? | Não: o sistema citou, com a página certa, o artigo de uma lei revogada |

Os detalhes, as ressalvas e a resposta à pergunta de pesquisa estão no
[artigo](docs/article/ThingIndex-artigo.pdf).

## Anexos

Os diagramas, as telas e as tabelas completas ficam fora do artigo, por causa do limite de
páginas, e estão em [`anexos/`](anexos/README.md):

| Anexo | Conteúdo |
|---|---|
| [A1](anexos/A1-requisitos-funcionais.pdf) | Requisitos funcionais e não funcionais |
| [A2](anexos/A2-casos-de-uso.pdf) | Diagrama de casos de uso |
| [A3](anexos/A3-sequencia-envio-e-pergunta.pdf) | Diagrama de sequência: envio de um documento e uma pergunta |
| [A4](anexos/A4-esquema-proveniencia.pdf) | Tabelas que ligam cada fonte ao documento e à página |
| [A5](anexos/A5-fluxo-de-recuperacao.pdf) | Fluxo de recuperação de contexto |
| [A6](anexos/A6-telas-do-prototipo.pdf) | Telas do sistema |
| [A7](anexos/A7-tabelas-completas-de-recuperacao.pdf) | Tabelas completas do teste de busca (corpus de controle C1) |
| [A8](anexos/A8-alteracoes-pos-medicao.pdf) | Correções feitas depois de medir, com o motivo e o efeito |
| [A9](anexos/A9-esquema-do-banco.pdf) | Esquema completo do banco (25 tabelas) |
| [A10](anexos/A10-acervos/README.md) | Acervos da avaliação: os PDFs indexados e as perguntas de cada um |

## Como funciona

1. O usuário envia um PDF. A API grava o arquivo e, na mesma transação, o documento e o
   pedido de indexação (*outbox*). Um despachante publica o pedido no RabbitMQ.
2. O *indexer* extrai páginas, parágrafos com posição, tabelas e sumário, e grava cada
   parte numa tabela, com o documento e a página.
3. Na pergunta, a API carrega as partes da coleção, calcula a nota BM25 e escolhe até 9
   fontes, cada uma com documento e página lidos do banco.
4. O LLM recebe a pergunta e essas fontes, com a instrução de responder só a partir delas.
5. A tela mostra a resposta e abre cada fonte no PDF, com o parágrafo destacado.

| Parte | Tecnologia |
|---|---|
| Frontend | Next.js 16 e React 19 |
| API e indexer | Go, com Gin e GORM |
| Banco | PostgreSQL 16 |
| Fila | RabbitMQ |
| Arquivos | armazenamento compatível com S3 (MinIO no ambiente local) |
| LLM | provedor configurável (os experimentos usaram Ollama) |

## Como executar

Pré-requisitos: Docker e Docker Compose v2.

```bash
cp .env.example .env        # ajuste as variáveis, se precisar
./scripts/bootstrap.sh
./scripts/stack.sh up       # ou ./scripts/stack.sh dev, com hot reload
./scripts/doctor.sh         # confere se os serviços estão de pé
```

| Serviço | Endereço |
|---|---|
| Frontend | http://localhost:3000 |
| API | http://localhost:8000 |
| Documentação da API | http://localhost:8000/docs |
| Console do MinIO | http://localhost:9001 |
| RabbitMQ | http://localhost:15672 |
| PostgreSQL | localhost:5434 |

Depois de criar uma conta, cadastre um provedor de LLM em **Configurações** (modelo, URL e,
se quiser acompanhar o custo, o preço por milhão de *tokens*), crie uma coleção e envie os
PDFs.

Para rodar a API e o frontend fora do Docker, suba só a infraestrutura com
`./scripts/stack.sh core` e, em terminais separados, `cd api && go run ./cmd/api` e
`cd frontend && pnpm install && pnpm dev`. Nesse caso, aponte `DB_HOST`, `DB_PORT` e
`MINIO_ENDPOINT` do `.env` para `localhost`.

## Testes

```bash
./scripts/test.sh unit                  # unitários do backend e do frontend
./scripts/test.sh backend-integration   # contra PostgreSQL, RabbitMQ e MinIO reais
./scripts/test.sh frontend-e2e          # interface no navegador (Playwright)
./scripts/teste-caos.sh                 # derruba fila, banco e armazenamento com o sistema rodando
```

## Reproduzir o artigo

- Resultados brutos, uma linha por pergunta e repetição: [`docs/article/results/`](docs/article/results/)
- Gabaritos de cada acervo, travados por *hash*: [`api/internal/evalcorpus/testdata/`](api/internal/evalcorpus/testdata/)
- Fonte LaTeX do artigo: [`docs/article/artigo.tex`](docs/article/artigo.tex)
- Os números do texto são conferidos contra os resultados por `scripts/check-article-numbers.py`,
  e os anexos são gerados por `scripts/gerar-anexos.py`.

## Estrutura do repositório

```text
api/          API, indexer e ferramentas de avaliação (Go)
frontend/     interface web (Next.js)
anexos/       anexos do artigo, em PDF
docs/article/ artigo, resultados, tabelas e imagens
examples/     PDFs usados na avaliação
scripts/      execução, testes, experimentos e conferência do artigo
```

## Licença

[MIT](LICENSE).
