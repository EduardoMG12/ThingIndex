<div align="center">
  <img src="../assets/palmas-horizontal-1.png" alt="IFPR Campus Palmas" width="300" />
</div>

<div align="center">
  <h1>LegisGraph: Crawler (Golang)</h1>
  <p><b>Course / Curso:</b> Sistemas de Informação</p>
  <p><b>Author / Autor:</b> Charles Eduardo Mello Guimarães</p>
  <p><b>Trabalho de Conclusão de Curso (TCC)</b></p>
</div>

<br />

<details>
<summary>🇺🇸 <b>English Version / Versão em Inglês</b></summary>

## 🚀 Features
This component is responsible for crawling Brazilian federal legislation from the `Planalto.gov.br` portal.
- **Format Handling**: Downloads both HTML and PDF versions of laws.
- **Hierarchical Context Tracking**: Tracks exactly where a link was found (Outgoing Context) and propagates the context of the link that led to the current page (Incoming Context).
- **Deduplication**: Uses a specialized Bloom Filter and content hashing to avoid re-downloading previously processed files, preventing cyclic infinite loops.

## 🐇 Queue & Triggering (RabbitMQ)
The Crawler operates strictly as a background worker (Consumer) using **RabbitMQ**. It listens to the `crawler_tasks` queue. It accepts a JSON payload containing the starting URL and requested maximum depth.

### 🧹 The `__PURGE__` Command
If you need to instantly stop the crawler, wipe the queue, and reset the internal memory caches without restarting the container, you can dispatch a payload setting the URL exactly to `"url": "__PURGE__"`.

## 🛠️ Stack & Execution
- **Language**: Golang (Go 1.21+)
- **Mensageria**: RabbitMQ (`amqp091-go`)
- **Containers**: Multi-stage Docker deployment for small footprint

Run via centralized Docker Compose, or locally using `go run cmd/crawler/main.go`.
</details>

<details>
<summary>🇧🇷 <b>Portuguese Version / Versão em Português</b></summary>

## 🚀 Funcionalidades
Este módulo é responsável pela varredura e download das leis federais a partir do site governamental `Planalto.gov.br`.
- **Extração Híbrida**: Permite o salvamento tanto dos documentos `.htm` como dos documentos em `.pdf`.
- **Rastreabilidade de Contexto Hierárquico**: Identifica e armazena com precisão de qual trecho do texto base (ex: "Art. 5º da CF88") determinado link surgiu.
- **Filtro contra Repetição**: Previne os conhecidos "ciclos infinitos judiciais" no processo web-scraping através de Bloom Filters em memória e de Hashing em disco, garantindo que nenhum documento rodará duplamente.

## 🐇 Mensageria e RabbitMQ
O Crawler adota uma mecânica fortemente "worker" (Consumidor), rodando em background aguardando mensagens na fila Rabbit. A **API Python** serve como intermediária para disparar tarefas via REST pra ele.

### 🧹 O Comando Mágico `__PURGE__`
Por sua complexidade, muitas vezes precisamos limpar rapidamente o cache dele com RabbitMQ. Injetando a URL especial `"url": "__PURGE__"`, ele fará um wipe completo da sua memória interna e da fila sem a necessidade de um reinício drástico de Container Docker.

## 🛠️ Tecnologias e Execução
- **Linguagem**: Golang (Excelente na gerência de concorrência com Goroutines)
- **Mensageria**: RabbitMQ (`amqp091-go`)
- **Container**: Foi criado um _Dockerfile Multi-stage_ de altíssima compactação e isolamento para servir junto aos outros recursos com o `docker compose up -d crawler`.
</details>

