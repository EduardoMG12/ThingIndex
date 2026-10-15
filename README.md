<div align="center">

<img src="assets/palmas-horizontal-1.png" alt="IFPR Campus Palmas" width="280" />

# ThingIndex

### Inteligência Documental com Proveniência Verificável sobre Modelo Relacional

Trabalho de Conclusão de Curso · Sistemas de Informação · IFPR Campus Palmas · 2026<br/>
**Charles Eduardo Mello Guimarães**

<br/>

<a href="docs/article/ThingIndex-artigo.pdf"><img src="https://img.shields.io/badge/Ler_o_artigo-PDF-387330?style=for-the-badge&logo=adobeacrobatreader&logoColor=white" alt="Ler o artigo" /></a>
<a href="#anexos"><img src="https://img.shields.io/badge/Anexos-A1_a_A10-009F48?style=for-the-badge" alt="Anexos" /></a>
<a href="#como-executar"><img src="https://img.shields.io/badge/Executar-Docker_Compose-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Executar" /></a>

<img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go" />
<img src="https://img.shields.io/badge/Next.js-16-000000?logo=next.js&logoColor=white" alt="Next.js" />
<img src="https://img.shields.io/badge/PostgreSQL-16-336791?logo=postgresql&logoColor=white" alt="PostgreSQL" />
<img src="https://img.shields.io/badge/RabbitMQ-fila-FF6600?logo=rabbitmq&logoColor=white" alt="RabbitMQ" />
<img src="https://img.shields.io/badge/licen%C3%A7a-MIT-green" alt="Licença MIT" />

</div>

---

O **ThingIndex** responde perguntas sobre coleções de PDFs, como leis, normas e manuais, e
mostra **de onde veio cada resposta**: o documento e a página, com o parágrafo destacado no
próprio PDF.

Cada parte do documento (página, parágrafo, linha de tabela, seção do sumário) é uma linha
de tabela no PostgreSQL, ligada ao documento por chave estrangeira e com a página de origem.
A busca é por palavras (BM25), sem banco vetorial. O LLM só entra no fim, para redigir a
resposta a partir das fontes que a busca escolheu. Por isso a página mostrada ao usuário vem
do banco, e não do texto escrito pelo modelo.

> **Pergunta de pesquisa:** *é possível sustentar um sistema de inteligência documental com
> auditabilidade utilizando exclusivamente modelagem relacional, sem depender de bancos
> vetoriais ou grafos como camada primária de recuperação?*

<p align="center">
  <img src="docs/article/anexos/telas/workspace.png" alt="Workspace do ThingIndex" width="920" />
  <br/>
  <sub>Workspace de uma coleção de normas de segurança do trabalho: chat com as fontes numeradas, PDF na página citada com o parágrafo destacado e resumo do documento.</sub>
</p>

## Sumário

- [O artigo](#o-artigo)
- [Resultados](#resultados)
- [Anexos](#anexos)
- [O sistema em telas](#o-sistema-em-telas)
- [Como funciona](#como-funciona)
- [Como executar](#como-executar)
- [Testes](#testes)
- [Reproduzir o artigo](#reproduzir-o-artigo)
- [Estrutura do repositório](#estrutura-do-repositório)
- [Como citar](#como-citar)

## O artigo

<p align="center">
  <a href="docs/article/ThingIndex-artigo.pdf">
    <img src="assets/readme/artigo.png" alt="Primeiras páginas do artigo" width="900" />
  </a>
  <br/>
  <a href="docs/article/ThingIndex-artigo.pdf"><b>📄 Abrir o artigo completo (PDF, 25 páginas)</b></a>
</p>

O trabalho segue a *Design Science Research*: o ThingIndex foi construído e avaliado em sete
acervos em português, seis de PDFs públicos reais (documentos do IFPR, leis, Normas
Regulamentadoras, manuais e um edital) e um de controle, com cerca de 870 páginas no total.
As hipóteses foram registradas antes de medir, os gabaritos ficam travados por *hash* e
toda correção feita depois da medição está registrada no [Anexo A8](anexos/A8-alteracoes-pos-medicao.pdf).

A fonte LaTeX está em [`docs/article/artigo.tex`](docs/article/artigo.tex).

## Resultados

<table>
  <tr>
    <td align="center" width="25%"><h2>69,5%</h2>de acerto em perguntas novas<br/><sub>cerca de 7 em 10 respostas</sub></td>
    <td align="center" width="25%"><h2>88,7%</h2>com a expansão da pergunta<br/><sub>14 perguntas ganhas, nenhuma perdida</sub></td>
    <td align="center" width="25%"><h2>≈ 1/26</h2>dos <i>tokens</i> de mandar o PDF inteiro<br/><sub>72 de 72 perguntas certas</sub></td>
    <td align="center" width="25%"><h2>13 × 11</h2>busca por palavras × por significado<br/><sub>de 16 consultas, sem diferença detectável</sub></td>
  </tr>
</table>

| Pergunta | Resultado |
|---|---|
| Busca por palavras ou por significado: qual acha mais? | Ficaram próximas: 13 de 16 contra 11 de 16 consultas, sem diferença detectável |
| Buscar trechos custa menos que mandar o PDF inteiro ao modelo? | Sim: na mediana, cerca de 1/26 dos *tokens*, com 72 de 72 perguntas certas contra 71 de 72 |
| Quanto o sistema acerta em perguntas novas? | 69,5% das respostas, cerca de 7 em 10 |
| A redação da pergunta muda o acerto? | Sim: 88,2% com as palavras do documento e 41,9% com outras palavras |
| Expandir a pergunta com um LLM ajuda? | De 69,5% para 88,7% (experimental, desligada por padrão) |
| Fonte certa garante resposta certa? | Não: o sistema citou, com a página certa, o artigo de uma lei revogada |

As ressalvas de cada número e a resposta à pergunta de pesquisa, condição por condição,
estão nas Seções 8 e 9 do artigo.

## Anexos

Os diagramas, as telas e as tabelas completas ficam fora do artigo, por causa do limite de
páginas. Clique em um anexo para abrir o PDF.

<table>
  <tr>
    <td align="center" width="20%"><a href="anexos/A1-requisitos-funcionais.pdf"><img src="assets/readme/A1.png" alt="A1" /></a><br/><b>A1</b><br/><sub>Requisitos funcionais e não funcionais</sub></td>
    <td align="center" width="20%"><a href="anexos/A2-casos-de-uso.pdf"><img src="assets/readme/A2.png" alt="A2" /></a><br/><b>A2</b><br/><sub>Casos de uso</sub></td>
    <td align="center" width="20%"><a href="anexos/A3-sequencia-envio-e-pergunta.pdf"><img src="assets/readme/A3.png" alt="A3" /></a><br/><b>A3</b><br/><sub>Sequência: envio de um documento e uma pergunta</sub></td>
    <td align="center" width="20%"><a href="anexos/A4-esquema-proveniencia.pdf"><img src="assets/readme/A4.png" alt="A4" /></a><br/><b>A4</b><br/><sub>Tabelas que ligam cada fonte ao documento e à página</sub></td>
    <td align="center" width="20%"><a href="anexos/A5-fluxo-de-recuperacao.pdf"><img src="assets/readme/A5.png" alt="A5" /></a><br/><b>A5</b><br/><sub>Fluxo de recuperação</sub></td>
  </tr>
  <tr>
    <td align="center"><a href="anexos/A6-telas-do-prototipo.pdf"><img src="assets/readme/A6.png" alt="A6" /></a><br/><b>A6</b><br/><sub>Telas do sistema</sub></td>
    <td align="center"><a href="anexos/A7-tabelas-completas-de-recuperacao.pdf"><img src="assets/readme/A7.png" alt="A7" /></a><br/><b>A7</b><br/><sub>Tabelas completas do teste de busca</sub></td>
    <td align="center"><a href="anexos/A8-alteracoes-pos-medicao.pdf"><img src="assets/readme/A8.png" alt="A8" /></a><br/><b>A8</b><br/><sub>Correções feitas depois de medir</sub></td>
    <td align="center"><a href="anexos/A9-esquema-do-banco.pdf"><img src="assets/readme/A9.png" alt="A9" /></a><br/><b>A9</b><br/><sub>Esquema completo do banco (25 tabelas)</sub></td>
    <td align="center"><a href="anexos/A10-acervos/README.md"><img src="assets/readme/A10.png" alt="A10" /></a><br/><b>A10</b><br/><sub>Acervos: os PDFs indexados e as perguntas</sub></td>
  </tr>
</table>

O A10 é uma pasta por acervo, com os PDFs que o sistema indexou e um `perguntas.pdf` com
todas as perguntas, a resposta esperada e a página onde ela está:
[controle C1](anexos/A10-acervos/01-controle-c1/) ·
[ensino](anexos/A10-acervos/02-ensino/) ·
[legislação](anexos/A10-acervos/03-legislacao/) ·
[segurança do trabalho](anexos/A10-acervos/04-seguranca-do-trabalho/) ·
[produto e consumidor](anexos/A10-acervos/05-produto-e-consumidor/) ·
[licitações](anexos/A10-acervos/06-licitacoes/) ·
[edital municipal](anexos/A10-acervos/07-edital-municipal/).

## O sistema em telas

<table>
  <tr>
    <td width="50%"><img src="docs/article/anexos/telas/04-area-de-trabalho-documentos.png" alt="Área de trabalho" /><br/><sub><b>Área de trabalho</b> · documentos indexados da coleção</sub></td>
    <td width="50%"><img src="docs/article/anexos/telas/06-area-de-trabalho-pdf.png" alt="PDFs lado a lado" /><br/><sub><b>Leitura</b> · PDFs da coleção abertos lado a lado</sub></td>
  </tr>
  <tr>
    <td><img src="docs/article/anexos/telas/07-operacoes.png" alt="Operações" /><br/><sub><b>Operações</b> · andamento da indexação</sub></td>
    <td><img src="docs/article/anexos/telas/09-auditoria.png" alt="Auditoria" /><br/><sub><b>Auditoria</b> · trilha de eventos dos documentos</sub></td>
  </tr>
  <tr>
    <td><img src="docs/article/anexos/telas/10-custos.png" alt="Custos" /><br/><sub><b>Custos</b> · consumo de <i>tokens</i> e custo estimado</sub></td>
    <td><img src="docs/article/anexos/telas/13-configuracoes-llm.png" alt="Provedor de LLM" /><br/><sub><b>Provedores</b> · modelo, URL e preço por milhão de <i>tokens</i></sub></td>
  </tr>
</table>

Todas as telas estão no [Anexo A6](anexos/A6-telas-do-prototipo.pdf).

## Como funciona

```mermaid
flowchart LR
    U(["Pergunta do usuário"]) --> B["Busca BM25<br/>nas tabelas da coleção"]
    B --> F["Até 9 fontes<br/>documento e página lidos do banco"]
    F --> L["LLM redige a resposta<br/>só a partir das fontes"]
    L --> T(["Tela abre cada fonte no PDF<br/>com o parágrafo destacado"])
```

O modelo não escolhe as fontes: elas vêm da busca e do banco antes de ele escrever.

```mermaid
flowchart LR
    FE["Frontend<br/>Next.js"] --> API["API<br/>Go"]
    API --> PG[("PostgreSQL 16<br/>estrutura dos PDFs")]
    API --> S3[("Armazenamento S3<br/>MinIO")]
    API -- "outbox" --> MQ["RabbitMQ"]
    MQ --> IX["Indexer<br/>Go"]
    IX --> PG
    API --> LLM["Provedor de LLM"]
```

No envio, a API grava o documento e o pedido de indexação na mesma transação (*outbox*); um
despachante publica o pedido no RabbitMQ, e o *indexer* extrai páginas, parágrafos com
posição, tabelas e sumário. Um teste de caos derruba a fila, o banco e o armazenamento com o
sistema em uso e confere que ele volta sozinho.

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
para acompanhar o custo, o preço por milhão de *tokens*), crie uma coleção e envie os PDFs.

<details>
<summary><b>Rodar a API e o frontend fora do Docker</b></summary>

<br/>

Suba só a infraestrutura e rode cada parte num terminal:

```bash
./scripts/stack.sh core
cd api && go run ./cmd/api
cd frontend && pnpm install && pnpm dev
```

No `.env`, aponte `DB_HOST`, `DB_PORT` e `MINIO_ENDPOINT` para `localhost`. Requer Go 1.26,
Node.js 20 e `pnpm`.

</details>

## Testes

```bash
./scripts/test.sh unit                  # unitários do backend e do frontend
./scripts/test.sh backend-integration   # contra PostgreSQL, RabbitMQ e MinIO reais
./scripts/test.sh frontend-e2e          # interface no navegador (Playwright)
./scripts/teste-caos.sh                 # derruba fila, banco e armazenamento com o sistema rodando
```

## Reproduzir o artigo

| O quê | Onde |
|---|---|
| Resultados brutos, uma linha por pergunta e repetição | [`docs/article/results/`](docs/article/results/) |
| Gabaritos de cada acervo, travados por *hash* | [`api/internal/evalcorpus/testdata/`](api/internal/evalcorpus/testdata/) |
| Tabelas geradas a partir dos resultados | [`docs/article/tables/`](docs/article/tables/) |
| Conferência dos números do texto contra os resultados | [`scripts/check-article-numbers.py`](scripts/check-article-numbers.py) |
| Geração dos anexos | [`scripts/gerar-anexos.py`](scripts/gerar-anexos.py) |

## Estrutura do repositório

```text
api/           API, indexer e ferramentas de avaliação (Go)
frontend/      interface web (Next.js)
anexos/        anexos do artigo, em PDF
docs/article/  artigo, resultados, tabelas e imagens
examples/      PDFs usados na avaliação
scripts/       execução, testes, experimentos e conferência do artigo
```

## Como citar

```bibtex
@misc{guimaraes2026thingindex,
  author       = {Guimarães, Charles Eduardo Mello},
  title        = {ThingIndex: Inteligência Documental com Proveniência Verificável sobre Modelo Relacional},
  howpublished = {Trabalho de Conclusão de Curso (Sistemas de Informação), Instituto Federal do Paraná, Campus Palmas},
  year         = {2026},
  url          = {https://github.com/EduardoMG12/ThingIndex}
}
```

## Licença

[MIT](LICENSE).
