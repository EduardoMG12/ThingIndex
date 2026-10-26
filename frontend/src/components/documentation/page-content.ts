import type { AppLocale } from "@/lib/i18n/config";

export type DocumentationTocItem = {
    id: string;
    label: string;
};

export type DocumentationMetricSnapshot = {
    label: string;
    value: string;
};

export type DocumentationPrinciple = {
    title: string;
    body: string;
};

export type DocumentationLayer = {
    label: string;
    description: string;
};

export type DocumentationStep = {
    step: string;
    title: string;
    description: string;
};

export type DocumentationSurface = {
    title: string;
    body: string;
};

export type DocumentationReferenceTrack = {
    label: string;
    references: string[];
};

export type DocumentationPageContent = {
    heroBadges: string[];
    heroTitle: {
        start: string;
        highlight: string;
    };
    heroDescription: string;
    heroSnapshot: DocumentationMetricSnapshot[];
    toc: DocumentationTocItem[];
    overview: {
        title: string;
        paragraphs: string[];
    };
    platformSurfaces: {
        title: string;
        intro: string;
        items: DocumentationSurface[];
    };
    productFocusTitle: string;
    principles: {
        title: string;
        items: DocumentationPrinciple[];
    };
    architecture: {
        title: string;
        intro: string;
        layersTitle: string;
        layers: DocumentationLayer[];
        goalsTitle: string;
        goals: string[];
        snapshot: string;
    };
    processing: {
        title: string;
        intro: string;
        steps: DocumentationStep[];
    };
    dataModel: {
        title: string;
        paragraphs: string[];
    };
    retrieval: {
        title: string;
        paragraphs: string[];
    };
    indexingBenchmarkTitle: string;
    providersTitle: string;
    security: {
        title: string;
        paragraphs: string[];
    };
    quality: {
        title: string;
        intro: string;
        items: string[];
        note: string;
    };
    deployment: {
        title: string;
        paragraphs: string[];
    };
    openSourceTitle: string;
    roadmap: {
        title: string;
        items: string[];
    };
    academic: {
        title: string;
        paragraphs: string[];
    };
    contributing: {
        title: string;
        paragraphs: string[];
    };
    references: {
        title: string;
        intro: string;
        tracks: DocumentationReferenceTrack[];
    };
};

const ARCHITECTURE_SNAPSHOT_EN = `[Client Workspace]
  -> auth / dashboard / documents / settings / docs

[Go API]
  -> auth
  -> uploads + metadata
  -> collection context
  -> provider management
  -> public runtime meta

[Processing]
  -> outbox dispatch
  -> RabbitMQ queue
  -> indexer worker
  -> structured extraction

[Persistence]
  -> PostgreSQL
  -> object storage
  -> pricing and audit records`;

const ARCHITECTURE_SNAPSHOT_PT_BR = `[Workspace do cliente]
  -> autenticacao / dashboard / documentos / configuracoes / docs

[API Go]
  -> autenticacao
  -> uploads + metadados
  -> contexto por colecao
  -> gestao de provedores
  -> metadados publicos do runtime

[Processamento]
  -> despacho por outbox
  -> fila RabbitMQ
  -> worker de indexacao
  -> extracao estruturada

[Persistencia]
  -> PostgreSQL
  -> armazenamento de objetos
  -> registros de pricing e auditoria`;

const ARCHITECTURE_SNAPSHOT_ES = `[Workspace del cliente]
  -> autenticacion / dashboard / documentos / configuracion / docs

[API Go]
  -> autenticacion
  -> cargas + metadatos
  -> contexto por coleccion
  -> gestion de providers
  -> metadatos publicos del runtime

[Procesamiento]
  -> despacho por outbox
  -> cola RabbitMQ
  -> worker de indexacion
  -> extraccion estructurada

[Persistencia]
  -> PostgreSQL
  -> almacenamiento de objetos
  -> registros de pricing y auditoria`;

function getPtBrContent(): DocumentationPageContent {
    return {
        heroBadges: [
            "documentação oficial",
            "runtime atual",
            "pesquisa separada do produto",
        ],
        heroTitle: {
            start: "Documentação técnica para um projeto",
            highlight: "robusto, auditável e de código aberto",
        },
        heroDescription:
            "O ThingIndex evoluiu para um sistema documental completo: autenticação, configuração de provedores, upload, indexação estrutural assíncrona, observabilidade, auditoria, chat grounded e documentação pública alinhada ao contrato real da API.",
        heroSnapshot: [
            { label: "Escopo atual", value: "Plataforma documental relacional com contexto consolidado" },
            { label: "Stack principal", value: "Next.js, Go, PostgreSQL, MinIO e RabbitMQ" },
            { label: "Operação", value: "Indexação assíncrona, auditoria e visibilidade de custo" },
            { label: "Abertura", value: "API pública, OpenAPI versionado e onboarding de código aberto" },
        ],
        toc: [
            { id: "overview", label: "Visão geral" },
            { id: "product-focus", label: "Foco do produto" },
            { id: "platform-surfaces", label: "Superfícies da plataforma" },
            { id: "principles", label: "Princípios de projeto" },
            { id: "architecture", label: "Arquitetura do sistema" },
            { id: "processing", label: "Ciclo de processamento" },
            { id: "data-model", label: "Modelo de dados" },
            { id: "retrieval", label: "Recuperação de contexto" },
            { id: "indexing-benchmark", label: "Benchmark de indexação" },
            { id: "providers", label: "Provedores e portabilidade" },
            { id: "security", label: "Segurança e governança" },
            { id: "quality", label: "Qualidade de engenharia" },
            { id: "deployment", label: "Deploy e operação" },
            { id: "open-source", label: "Onboarding de código aberto" },
            { id: "roadmap", label: "Roadmap atual" },
            { id: "academic", label: "TCC e pesquisa" },
            { id: "contributing", label: "Contribuição" },
            { id: "references", label: "Referências" },
        ],
        overview: {
            title: "Visão geral",
            paragraphs: [
                "O ThingIndex não é mais apenas uma prova de conceito de ingestão documental. O repositório atual entrega uma plataforma relacional completa para organizar coleções, processar documentos, preservar estrutura e recuperar contexto com rastreabilidade.",
                "Para fins de TCC e visibilidade pública, a premissa desta página é simples: documentar somente o que o sistema já entrega, marcar pesquisa e backlog como trilhas separadas e manter a narrativa técnica coerente com o monorepo.",
            ],
        },
        platformSurfaces: {
            title: "Superfícies atuais da plataforma",
            intro: "O produto hoje já expõe camadas distintas de operação, observabilidade e configuração. A documentação precisa refletir esse escopo com clareza.",
            items: [
                {
                    title: "Dashboard e ingestão",
                    body: "Entrada principal para categorias, coleções, envio de arquivos e organização do fluxo documental.",
                },
                {
                    title: "Biblioteca documental",
                    body: "Leitura operacional de documentos, visualização, indexação estruturada, anomalias e auditoria.",
                },
                {
                    title: "Configurações e provedores",
                    body: "Configuração segura de armazenamento e LLM, verificações de prontidão, testes de conexão e papéis separados para indexação e chat.",
                },
                {
                    title: "Código aberto e contrato HTTP",
                    body: "README, OpenAPI, exemplos HTTP, benchmark público e matrizes de suporte para consumo externo.",
                },
            ],
        },
        productFocusTitle: "Foco do produto",
        principles: {
            title: "Princípios de projeto",
            items: [
                {
                    title: "Relacional em primeiro lugar",
                    body: "Coleções, categorias, documentos e estruturas tipadas formam o núcleo do sistema.",
                },
                {
                    title: "Borda HTTP fina",
                    body: "Handlers cuidam de validação, autenticação e serialização; regras ficam em casos de uso e repositórios.",
                },
                {
                    title: "Runtime observável",
                    body: "Auditoria, benchmark público, readiness e telemetria aparecem como partes do produto, não como pós-venda.",
                },
                {
                    title: "Escopo honesto",
                    body: "Implementação atual, pesquisa derivada e roadmap ficam explicitamente separados.",
                },
            ],
        },
        architecture: {
            title: "Arquitetura do sistema",
            intro: "A arquitetura atual separa interface, API, processamento assíncrono e persistência para reduzir acoplamento e sustentar evolução incremental.",
            layersTitle: "Camadas",
            layers: [
                { label: "Interface web", description: "autenticação, dashboard, biblioteca documental, configurações, custo e documentação." },
                { label: "API", description: "upload, provedores, contexto por coleção, benchmark e metadados públicos." },
                { label: "Processamento", description: "outbox, filas, worker de indexação e enriquecimento opcional." },
                { label: "Persistência", description: "PostgreSQL relacional, armazenamento de objetos e registros de auditoria/telemetria." },
            ],
            goalsTitle: "Objetivos arquiteturais",
            goals: [
                "Preservar proveniência com joins previsíveis.",
                "Permitir troca de provedores sem reescrever o núcleo do produto.",
                "Separar fluxo síncrono de interface e pipeline assíncrono de indexação.",
                "Expor sinais públicos suficientes para onboarding, operação e defesa técnica.",
            ],
            snapshot: ARCHITECTURE_SNAPSHOT_PT_BR,
        },
        processing: {
            title: "Ciclo de processamento",
            intro: "O pipeline principal já está fechado do upload até a consulta contextualizada, com separação entre persistência inicial e processamento estrutural posterior.",
            steps: [
                { step: "01", title: "Upload e registro", description: "O documento é validado, vinculado a categoria e coleção e persistido no armazenamento." },
                { step: "02", title: "Despacho assíncrono", description: "Eventos de outbox e filas desacoplam a interface do custo de indexação." },
                { step: "03", title: "Extração estrutural", description: "Parágrafos, tabelas, estrutura e metadados são persistidos em estruturas tipadas." },
                { step: "04", title: "Contexto consolidado", description: "A API recompõe coleção, documentos e estruturas para navegação e chat grounded." },
                { step: "05", title: "Observabilidade", description: "Resumo agregado, benchmark, readiness e auditoria deixam o runtime mais legível para operação." },
            ],
        },
        dataModel: {
            title: "Modelo de dados",
            paragraphs: [
                "O modelo relacional atual organiza o domínio em `categories`, `collections`, `documents`, `paragraph_nodes`, `table_sections` e `metadata_pairs`, preservando ordem, contexto e rastreabilidade.",
                "Esse desenho também sustenta auditoria, export de índice, monitoramento documental e leitura incremental na interface web sem depender de um núcleo vetorial para a navegação principal.",
            ],
        },
        retrieval: {
            title: "Recuperação de contexto",
            paragraphs: [
                "O diferencial implementado hoje é a recuperação relacional por coleção e documento. A API agrega estruturas persistidas e devolve uma visão consolidada, adequada para revisão, auditoria e resposta grounded.",
                "Comparações com GraphRAG, embeddings ou trilhas vetoriais permanecem válidas como pesquisa, mas não são vendidas aqui como mecanismo central do runtime atual.",
            ],
        },
        indexingBenchmarkTitle: "Benchmark de indexação",
        providersTitle: "Provedores e portabilidade",
        security: {
            title: "Segurança e governança",
            paragraphs: [
                "O runtime atual já inclui autenticação JWT, escopo por usuário, rotas protegidas, criptografia de credenciais de provedores, validação de uploads e proxies autenticados para conteúdo documental.",
                "Na frente de provedores, a API também impõe validações de configuração sensível, readiness estruturado e matrizes públicas de suporte para evitar promessas implícitas sobre caminhos ainda parciais.",
            ],
        },
        quality: {
            title: "Qualidade de engenharia",
            intro: "Como projeto de TCC com ambição de código aberto, a robustez precisa ser visível não só no código, mas também na forma de verificar o runtime.",
            items: [
                "testes unitários na interface web e no backend para fluxos críticos e helpers de domínio;",
                "testes smoke HTTP, integrações reais e cenários autenticados cobrindo upload, provedores, auditoria e chat grounded;",
                "benchmark público versionado para indexação com corpus controlado e critérios explícitos;",
                "README raiz, READMEs por camada, exemplos HTTP e documentação interativa servida pela implantação atual.",
            ],
            note: "A documentação passa a assumir explicitamente que qualidade e verificabilidade são parte do produto, não um anexo acadêmico.",
        },
        deployment: {
            title: "Deploy e operação",
            paragraphs: [
                "O ambiente local usa Docker Compose com interface web, API, PostgreSQL, MinIO e RabbitMQ. O runtime também conta com worker de indexação e worker de pricing como processos separados do servidor HTTP.",
                "Para quem consome o projeto de fora, a experiência agora cobre tanto o caminho auto-hospedado no monorepo quanto o consumo direto de uma API publicada com o mesmo contrato OpenAPI.",
            ],
        },
        openSourceTitle: "Onboarding de código aberto",
        roadmap: {
            title: "Roadmap atual",
            items: [
                "expandir validação concreta de provedores em implantações além do caminho principal atual;",
                "ampliar o corpus e os cenários avaliados no benchmark público de indexação;",
                "manter auditoria, onboarding e documentação sempre sincronizados com o runtime entregue.",
            ],
        },
        academic: {
            title: "TCC e pesquisa",
            paragraphs: [
                "O repositório é ao mesmo tempo artefato de engenharia e artefato acadêmico. Por isso, esta página separa implementação presente, pesquisa relacionada e evolução futura com a maior disciplina possível.",
                "Esse recorte fortalece a defesa técnica do TCC: o que está aqui documentado precisa ser demonstrável hoje no código, na interface ou na API pública.",
            ],
        },
        contributing: {
            title: "Contribuição",
            paragraphs: [
                "As contribuições mais valiosas são as que aumentam clareza e confiabilidade: testes, fixtures, documentação alinhada, melhorias incrementais de UX e validação mais concreta de provedores e fluxos documentais.",
                "Mudanças grandes continuam bem-vindas, mas precisam preservar a honestidade do escopo e a legibilidade do runtime principal.",
            ],
        },
        references: {
            title: "Referências",
            intro: "A leitura recomendada do repositório agora fica organizada por trilhas, para separar contrato atual, operação de código aberto e materiais de pesquisa.",
            tracks: [
                { label: "Sistema atual", references: ["README.md", "api/README.md", "frontend/README.md", "esta página"] },
                { label: "Contrato e operação", references: ["examples/http/thingindex-api.http", "api/docs/swagger.yaml", "docs/open-source-onboarding.md"] },
                { label: "Prontidão atual", references: ["docs/functional-readiness-spec.md"] },
                { label: "Pesquisa e continuidade", references: ["docs/proposta-continuacao-tcc.md", "docs/roadmap/", "docs/README.md"] },
            ],
        },
    };
}

function getEnContent(): DocumentationPageContent {
    return {
        heroBadges: [
            "official documentation",
            "current runtime",
            "research separated from product",
        ],
        heroTitle: {
            start: "Engineering documentation for a",
            highlight: "robust, auditable, open-source project",
        },
        heroDescription:
            "ThingIndex has evolved into a complete document runtime: authentication, provider configuration, uploads, asynchronous structured indexing, observability, audit, grounded chat, and public documentation aligned with the actual API contract.",
        heroSnapshot: [
            { label: "Current scope", value: "Relational document platform with consolidated context" },
            { label: "Core stack", value: "Next.js, Go, PostgreSQL, MinIO, and RabbitMQ" },
            { label: "Operations", value: "Async indexing, audit trails, and cost visibility" },
            { label: "Open access", value: "Public API, versioned OpenAPI, and open-source onboarding" },
        ],
        toc: [
            { id: "overview", label: "Overview" },
            { id: "product-focus", label: "Product focus" },
            { id: "platform-surfaces", label: "Platform surfaces" },
            { id: "principles", label: "Design principles" },
            { id: "architecture", label: "System architecture" },
            { id: "processing", label: "Processing lifecycle" },
            { id: "data-model", label: "Data model" },
            { id: "retrieval", label: "Context retrieval" },
            { id: "indexing-benchmark", label: "Indexing benchmark" },
            { id: "providers", label: "Providers and portability" },
            { id: "security", label: "Security and governance" },
            { id: "quality", label: "Engineering quality" },
            { id: "deployment", label: "Deployment and operations" },
            { id: "open-source", label: "Open-source onboarding" },
            { id: "roadmap", label: "Current roadmap" },
            { id: "academic", label: "Academic scope" },
            { id: "contributing", label: "Contributing" },
            { id: "references", label: "References" },
        ],
        overview: {
            title: "Overview",
            paragraphs: [
                "ThingIndex is no longer just a document-ingestion proof of concept. The current repository delivers a complete relational platform to organize collections, process documents, preserve structure, and recover context with traceability.",
                "For both academic review and public visibility, the rule of this page is simple: document only what the runtime already delivers, keep research and backlog explicitly separated, and make the technical narrative match the monorepo.",
            ],
        },
        platformSurfaces: {
            title: "Current platform surfaces",
            intro: "The product already exposes distinct layers for operation, observability, and configuration. The documentation needs to reflect that scope clearly.",
            items: [
                {
                    title: "Dashboard and ingestion",
                    body: "Main entry point for categories, collections, uploads, and the primary document workflow.",
                },
                {
                    title: "Document library",
                    body: "Operational reading of documents with preview, structured indexing, anomaly highlights, and audit trails.",
                },
                {
                    title: "Settings and providers",
                    body: "Secure storage and LLM configuration, readiness, connection tests, and separate roles for indexing and chat.",
                },
                {
                    title: "Open-source and HTTP contract",
                    body: "README, OpenAPI, HTTP examples, public benchmark, and support matrices for external consumers.",
                },
            ],
        },
        productFocusTitle: "Product focus",
        principles: {
            title: "Design principles",
            items: [
                {
                    title: "Relational first",
                    body: "Collections, categories, documents, and typed structures form the core of the system.",
                },
                {
                    title: "Thin HTTP boundary",
                    body: "Handlers own validation, auth, and serialization; business rules stay in use cases and repositories.",
                },
                {
                    title: "Observable runtime",
                    body: "Audit, public benchmarking, readiness, and telemetry are part of the product surface, not post-facto extras.",
                },
                {
                    title: "Honest scope",
                    body: "Current implementation, derived research, and future roadmap remain explicitly separated.",
                },
            ],
        },
        architecture: {
            title: "System architecture",
            intro: "The current architecture separates interface, API, asynchronous processing, and persistence to reduce coupling and support incremental evolution.",
            layersTitle: "Layers",
            layers: [
                { label: "Frontend", description: "authentication, dashboard, document library, settings, cost visibility, and docs." },
                { label: "API", description: "uploads, providers, collection context, benchmark endpoints, and public metadata." },
                { label: "Processing", description: "outbox, queues, indexing worker, and optional enrichment." },
                { label: "Persistence", description: "relational PostgreSQL, object storage, and audit/telemetry records." },
            ],
            goalsTitle: "Architectural goals",
            goals: [
                "Preserve provenance through predictable joins.",
                "Allow provider changes without rewriting the product core.",
                "Separate synchronous UI flows from the asynchronous indexing pipeline.",
                "Expose enough public signals for onboarding, operations, and technical defense.",
            ],
            snapshot: ARCHITECTURE_SNAPSHOT_EN,
        },
        processing: {
            title: "Processing lifecycle",
            intro: "The main pipeline is already closed from upload to contextual retrieval, with a clear separation between initial persistence and later structural processing.",
            steps: [
                { step: "01", title: "Upload and registration", description: "The document is validated, linked to category/collection, and persisted in storage." },
                { step: "02", title: "Asynchronous dispatch", description: "Outbox events and queues decouple the interface from indexing cost." },
                { step: "03", title: "Structured extraction", description: "Paragraphs, tables, outline, and metadata are persisted in typed structures." },
                { step: "04", title: "Consolidated context", description: "The API reconstructs collections, documents, and structures for navigation and grounded chat." },
                { step: "05", title: "Observability", description: "Aggregated monitoring, benchmark data, readiness, and audit make the runtime easier to operate." },
            ],
        },
        dataModel: {
            title: "Data model",
            paragraphs: [
                "The current relational model organizes the domain around `categories`, `collections`, `documents`, `paragraph_nodes`, `table_sections`, and `metadata_pairs`, preserving order, context, and traceability.",
                "That shape also supports audit, index export, document monitoring, and incremental frontend reading without relying on a vector-first core for the main navigation model.",
            ],
        },
        retrieval: {
            title: "Context retrieval",
            paragraphs: [
                "The implemented differentiator today is relational context retrieval by collection and document. The API aggregates persisted structures and returns a consolidated view suited for review, audit, and grounded answers.",
                "Comparisons with GraphRAG, embeddings, or vector-first paths remain useful as research, but they are not presented here as the main mechanism of the current runtime.",
            ],
        },
        indexingBenchmarkTitle: "Indexing benchmark",
        providersTitle: "Providers and portability",
        security: {
            title: "Security and governance",
            paragraphs: [
                "The current runtime already includes JWT authentication, per-user scope, protected routes, encrypted provider credentials, upload validation, and authenticated document proxies.",
                "On the provider side, the API also enforces sensitive-config validation, structured readiness, and public support matrices so that partial paths are not implied as production-ready.",
            ],
        },
        quality: {
            title: "Engineering quality",
            intro: "As both a TCC artifact and an open-source project, robustness needs to be visible not only in code but also in the way the runtime can be verified.",
            items: [
                "frontend and backend unit tests for critical flows and domain helpers;",
                "HTTP smokes, real integrations, and authenticated scenarios covering uploads, providers, audit, and grounded chat;",
                "a public, versioned indexing benchmark with a controlled corpus and explicit criteria;",
                "root and layer-specific READMEs, HTTP examples, and interactive documentation served by the current deployment.",
            ],
            note: "This documentation now treats quality and verifiability as part of the product surface, not as a separate academic appendix.",
        },
        deployment: {
            title: "Deployment and operations",
            paragraphs: [
                "The local environment uses Docker Compose with frontend, API, PostgreSQL, MinIO, and RabbitMQ. The runtime also includes separate indexing and pricing workers outside the HTTP server process.",
                "For external consumers, the experience now covers both the self-hosted monorepo path and direct consumption of a hosted API through the same OpenAPI contract.",
            ],
        },
        openSourceTitle: "Open-source onboarding",
        roadmap: {
            title: "Current roadmap",
            items: [
                "expand concrete provider validation in deployments beyond the current mainline path;",
                "broaden the public indexing benchmark corpus and citation-oriented scenarios;",
                "keep audit, onboarding, and documentation continuously synchronized with the shipped runtime.",
            ],
        },
        academic: {
            title: "Academic scope",
            paragraphs: [
                "The repository is both an engineering artifact and an academic artifact. For that reason, this page separates present implementation, related research, and future evolution as clearly as possible.",
                "That boundary strengthens the TCC defense: everything documented here should be demonstrable today in code, UI, or public API.",
            ],
        },
        contributing: {
            title: "Contributing",
            paragraphs: [
                "The most valuable contributions are the ones that improve clarity and confidence: tests, fixtures, aligned documentation, incremental UX refinements, and more concrete provider validation.",
                "Larger changes are still welcome, but they need to preserve honest scope and keep the main runtime legible.",
            ],
        },
        references: {
            title: "References",
            intro: "Recommended repository reading is now grouped into tracks to separate the current contract, open-source operations, and research material.",
            tracks: [
                { label: "Current runtime", references: ["README.md", "api/README.md", "frontend/README.md", "this page"] },
                { label: "Contract and operations", references: ["examples/http/thingindex-api.http", "api/docs/swagger.yaml", "docs/open-source-onboarding.md"] },
                { label: "Current readiness", references: ["docs/functional-readiness-spec.md"] },
                { label: "Research and continuation", references: ["docs/proposta-continuacao-tcc.md", "docs/roadmap/", "docs/README.md"] },
            ],
        },
    };
}

function getEsContent(): DocumentationPageContent {
    return {
        heroBadges: [
            "documentación oficial",
            "runtime actual",
            "investigación separada del producto",
        ],
        heroTitle: {
            start: "Documentación técnica para un proyecto",
            highlight: "robusto, auditable y open source",
        },
        heroDescription:
            "ThingIndex evolucionó hacia un runtime documental completo: autenticación, configuración de providers, carga, indexación estructural asíncrona, observabilidad, auditoría, chat grounded y documentación pública alineada con el contrato real de la API.",
        heroSnapshot: [
            { label: "Alcance actual", value: "Plataforma documental relacional con contexto consolidado" },
            { label: "Stack principal", value: "Next.js, Go, PostgreSQL, MinIO y RabbitMQ" },
            { label: "Operación", value: "Indexación asíncrona, auditoría y visibilidad de costos" },
            { label: "Apertura", value: "API pública, OpenAPI versionado y onboarding open source" },
        ],
        toc: [
            { id: "overview", label: "Visión general" },
            { id: "product-focus", label: "Foco del producto" },
            { id: "platform-surfaces", label: "Superficies de la plataforma" },
            { id: "principles", label: "Principios de diseño" },
            { id: "architecture", label: "Arquitectura del sistema" },
            { id: "processing", label: "Ciclo de procesamiento" },
            { id: "data-model", label: "Modelo de datos" },
            { id: "retrieval", label: "Recuperación de contexto" },
            { id: "indexing-benchmark", label: "Benchmark de indexación" },
            { id: "providers", label: "Providers y portabilidad" },
            { id: "security", label: "Seguridad y gobernanza" },
            { id: "quality", label: "Calidad de ingeniería" },
            { id: "deployment", label: "Deploy y operación" },
            { id: "open-source", label: "Onboarding open source" },
            { id: "roadmap", label: "Roadmap actual" },
            { id: "academic", label: "TCC e investigación" },
            { id: "contributing", label: "Contribución" },
            { id: "references", label: "Referencias" },
        ],
        overview: {
            title: "Visión general",
            paragraphs: [
                "ThingIndex ya no es solo una prueba de concepto de ingesta documental. El repositorio actual entrega una plataforma relacional completa para organizar collections, procesar documentos, preservar estructura y recuperar contexto con trazabilidad.",
                "Para revisión académica y visibilidad pública, la regla de esta página es simple: documentar solo lo que el runtime ya entrega, separar explícitamente investigación y backlog, y hacer que la narrativa técnica coincida con el monorepo.",
            ],
        },
        platformSurfaces: {
            title: "Superficies actuales de la plataforma",
            intro: "El producto ya expone capas distintas de operación, observabilidad y configuración. La documentación tiene que reflejar ese alcance con claridad.",
            items: [
                {
                    title: "Dashboard e ingesta",
                    body: "Punto principal de entrada para categorías, collections, cargas y el flujo documental principal.",
                },
                {
                    title: "Biblioteca documental",
                    body: "Lectura operativa de documentos con preview, indexación estructurada, señales de anomalía y auditoría.",
                },
                {
                    title: "Settings y providers",
                    body: "Configuración segura de storage y LLM, readiness, pruebas de conexión y roles separados para indexación y chat.",
                },
                {
                    title: "Open source y contrato HTTP",
                    body: "README, OpenAPI, ejemplos HTTP, benchmark público y matrices de soporte para consumo externo.",
                },
            ],
        },
        productFocusTitle: "Foco del producto",
        principles: {
            title: "Principios de diseño",
            items: [
                {
                    title: "Relational first",
                    body: "Collections, categorías, documentos y estructuras tipadas forman el núcleo del sistema.",
                },
                {
                    title: "Borde HTTP delgado",
                    body: "Los handlers resuelven validación, auth y serialización; las reglas viven en use cases y repositorios.",
                },
                {
                    title: "Runtime observable",
                    body: "Auditoría, benchmark público, readiness y telemetría forman parte de la superficie del producto.",
                },
                {
                    title: "Alcance honesto",
                    body: "La implementación actual, la investigación derivada y el roadmap futuro permanecen claramente separados.",
                },
            ],
        },
        architecture: {
            title: "Arquitectura del sistema",
            intro: "La arquitectura actual separa interfaz, API, procesamiento asíncrono y persistencia para reducir acoplamiento y sostener evolución incremental.",
            layersTitle: "Capas",
            layers: [
                { label: "Frontend", description: "autenticación, dashboard, biblioteca documental, settings, costos y documentación." },
                { label: "API", description: "cargas, providers, contexto por collection, benchmark y metadatos públicos." },
                { label: "Procesamiento", description: "outbox, colas, worker de indexación y enriquecimiento opcional." },
                { label: "Persistencia", description: "PostgreSQL relacional, object storage y registros de auditoría/telemetría." },
            ],
            goalsTitle: "Objetivos arquitectónicos",
            goals: [
                "Preservar proveniencia con joins previsibles.",
                "Permitir cambios de provider sin reescribir el núcleo del producto.",
                "Separar el flujo síncrono de interfaz del pipeline asíncrono de indexación.",
                "Exponer suficientes señales públicas para onboarding, operación y defensa técnica.",
            ],
            snapshot: ARCHITECTURE_SNAPSHOT_ES,
        },
        processing: {
            title: "Ciclo de procesamiento",
            intro: "El pipeline principal ya está cerrado desde la carga hasta la consulta contextualizada, con separación entre persistencia inicial y procesamiento estructural posterior.",
            steps: [
                { step: "01", title: "Carga y registro", description: "El documento se valida, se vincula a categoría/collection y se persiste en storage." },
                { step: "02", title: "Despacho asíncrono", description: "Eventos de outbox y colas desacoplan la interfaz del costo de indexación." },
                { step: "03", title: "Extracción estructural", description: "Párrafos, tablas, outline y metadatos se persisten en estructuras tipadas." },
                { step: "04", title: "Contexto consolidado", description: "La API recompone collections, documentos y estructuras para navegación y chat grounded." },
                { step: "05", title: "Observabilidad", description: "Monitoreo agregado, benchmark, readiness y auditoría hacen el runtime más operable." },
            ],
        },
        dataModel: {
            title: "Modelo de datos",
            paragraphs: [
                "El modelo relacional actual organiza el dominio alrededor de `categories`, `collections`, `documents`, `paragraph_nodes`, `table_sections` y `metadata_pairs`, preservando orden, contexto y trazabilidad.",
                "Ese diseño también sostiene auditoría, export de índice, monitoreo documental y lectura incremental en frontend sin depender de un núcleo vectorial para la navegación principal.",
            ],
        },
        retrieval: {
            title: "Recuperación de contexto",
            paragraphs: [
                "El diferencial implementado hoy es la recuperación relacional por collection y documento. La API agrega estructuras persistidas y devuelve una vista consolidada apta para revisión, auditoría y respuestas grounded.",
                "Las comparaciones con GraphRAG, embeddings o caminos vectoriales siguen siendo útiles como investigación, pero no se presentan aquí como mecanismo central del runtime actual.",
            ],
        },
        indexingBenchmarkTitle: "Benchmark de indexación",
        providersTitle: "Providers y portabilidad",
        security: {
            title: "Seguridad y gobernanza",
            paragraphs: [
                "El runtime actual ya incluye autenticación JWT, alcance por usuario, rutas protegidas, cifrado de credenciales de providers, validación de cargas y proxies autenticados para contenido documental.",
                "En providers, la API también impone validación de configuración sensible, readiness estructurado y matrices públicas de soporte para evitar que los caminos parciales parezcan listos para producción.",
            ],
        },
        quality: {
            title: "Calidad de ingeniería",
            intro: "Como artefacto de TCC y proyecto open source, la robustez tiene que ser visible no solo en el código, sino también en cómo se verifica el runtime.",
            items: [
                "tests unitarios en frontend y backend para flujos críticos y helpers de dominio;",
                "smokes HTTP, integraciones reales y escenarios autenticados cubriendo cargas, providers, auditoría y chat grounded;",
                "benchmark público y versionado de indexación con corpus controlado y criterios explícitos;",
                "README raíz, READMEs por capa, ejemplos HTTP y documentación interactiva servida por el deployment actual.",
            ],
            note: "Esta documentación pasa a tratar la calidad y la verificabilidad como parte de la superficie del producto, no como un apéndice académico.",
        },
        deployment: {
            title: "Deploy y operación",
            paragraphs: [
                "El entorno local usa Docker Compose con frontend, API, PostgreSQL, MinIO y RabbitMQ. El runtime también incluye workers separados de indexación y pricing fuera del proceso HTTP.",
                "Para consumidores externos, la experiencia ahora cubre tanto el camino self-hosted del monorepo como el consumo directo de una API publicada bajo el mismo contrato OpenAPI.",
            ],
        },
        openSourceTitle: "Onboarding open source",
        roadmap: {
            title: "Roadmap actual",
            items: [
                "expandir la validación concreta de providers en deployments más allá del camino principal actual;",
                "ampliar el corpus y los escenarios evaluados en el benchmark público de indexación;",
                "mantener auditoría, onboarding y documentación sincronizados con el runtime entregado.",
            ],
        },
        academic: {
            title: "TCC e investigación",
            paragraphs: [
                "El repositorio es al mismo tiempo un artefacto de ingeniería y un artefacto académico. Por eso esta página separa implementación presente, investigación relacionada y evolución futura con la mayor disciplina posible.",
                "Ese recorte fortalece la defensa técnica del TCC: todo lo documentado aquí debe poder demostrarse hoy en código, UI o API pública.",
            ],
        },
        contributing: {
            title: "Contribución",
            paragraphs: [
                "Las contribuciones más valiosas son las que aumentan claridad y confianza: tests, fixtures, documentación alineada, mejoras incrementales de UX y validación más concreta de providers.",
                "Los cambios grandes siguen siendo bienvenidos, pero deben preservar el alcance honesto y la legibilidad del runtime principal.",
            ],
        },
        references: {
            title: "Referencias",
            intro: "La lectura recomendada del repositorio ahora se agrupa en pistas para separar contrato actual, operación open source y material de investigación.",
            tracks: [
                { label: "Runtime actual", references: ["README.md", "api/README.md", "frontend/README.md", "esta página"] },
                { label: "Contrato y operación", references: ["examples/http/thingindex-api.http", "api/docs/swagger.yaml", "docs/open-source-onboarding.md"] },
                { label: "Prontitud actual", references: ["docs/functional-readiness-spec.md"] },
                { label: "Investigación y continuidad", references: ["docs/proposta-continuacao-tcc.md", "docs/roadmap/", "docs/README.md"] },
            ],
        },
    };
}

export function getDocumentationPageContent(locale: AppLocale): DocumentationPageContent {
    switch (locale) {
        case "pt-BR":
            return getPtBrContent();
        case "es":
            return getEsContent();
        default:
            return getEnContent();
    }
}
