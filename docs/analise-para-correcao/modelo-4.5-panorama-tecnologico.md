# Modelo de Texto — 4.5 Panorama Tecnológico

## Regra de Ouro

NÃO escreva definições de tecnologia como se fossem propaganda de site.
Escreva JUSTIFICATIVAS: "Optou-se pela tecnologia X porque Y, em vez de Z."

---

## Texto Modelo

### 4.5 Panorama Tecnológico

Esta subseção apresenta as tecnologias escolhidas para a implementação do ThingIndex e as justificativas para cada escolha. O critério central de seleção foi a adequação aos requisitos de auditabilidade, desempenho e manutenibilidade do sistema.

**Next.js.** O frontend do ThingIndex é construído com Next.js (framework React mantido pela Vercel). A escolha se deve a três fatores: (1) suporte nativo a Server Components, que permite renderizar páginas no servidor sem expor chaves de API ou lógica de negócio ao cliente; (2) App Router, que simplifica a estrutura de rotas e layouts; e (3) ecossistema maduro de componentes e ferramentas de teste. Em contraste com alternativas como Vue.js ou SvelteKit, o Next.js oferece o ecossistema mais consolidado para aplicações fullstack com React.

**Go.** A API e os workers de indexação são implementados em Go (Golang), linguagem criada na Google em 2009. Go foi escolhida por seu modelo de concorrência baseado em goroutines, que permite processar múltiplos documentos simultaneamente sem a complexidade de threads tradicionais. Além disso, a tipagem estática e a compilação para binário único facilitam a implantação e reduzem erros em tempo de execução. Alternativas como Node.js (tipagem dinâmica, event loop single-thread) ou Python (GIL, desempenho inferior) foram descartadas por não atenderem aos requisitos de concorrência do pipeline de indexação.

**PostgreSQL.** O banco de dados principal é o PostgreSQL (PostgreSQL Global Development Group, 1996), um sistema relacional maduro com suporte a JSONB, índices avançados e transações ACID. A escolha é consistente com a decisão arquitetural de priorizar consistência forte e auditabilidade. O pgvector, extensão que permite armazenamento e consulta de vetores, está disponível para futuras comparações com abordagens semânticas, mas não é utilizado na baseline relacional inicial.

**Object Storage (MinIO / S3).** Os arquivos brutos (PDFs) são armazenados em object storage compatível com a API S3, utilizando MinIO em desenvolvimento e Google Cloud Storage ou Cloudflare R2 em produção. Essa separação entre armazenamento de objetos (arquivos binários) e banco relacional (metadados e estrutura) é intencional: permite acesso aos arquivos via URLs temporárias (presigned URLs) sem sobrecarregar a API com proxy de bytes, e mantém o banco com tamanho gerenciável.

**Filas Assíncronas (RabbitMQ).** O pipeline de indexação é orquestrado por filas de mensagens utilizando RabbitMQ. O padrão **outbox** é empregado para garantir que nenhum evento de indexação seja perdido: quando um documento é persistido no banco, um evento de indexação é registrado na mesma transação. Um processo dispatcher lê os eventos da tabela de outbox e os publica no RabbitMQ. Caso a API falhe antes de publicar, o evento ainda está no banco e será republicado na próxima tentativa. O RabbitMQ foi escolhido por sua maturidade, suporte a filas de retry e dead-letter queues (DLQ), e documentação extensa.

**Docker Compose.** Todo o ambiente de desenvolvimento é reproduzível via Docker Compose, com healthchecks entre serviços. Isso garante que qualquer desenvolvedor possa executar o sistema com um único comando, independentemente do sistema operacional.

---

## Checklist

- [ ] Cada tecnologia tem uma JUSTIFICATIVA, não uma definição
- [ ] Comparei com alternativas (ex: Go vs Node.js vs Python)
- [ ] Não usei versões específicas (Go 1.25, Next.js 16 — isso envelhece mal)
- [ ] Conectei cada tecnologia a um requisito do ThingIndex
- [ ] O texto não parece lista de ferramentas de currículo
