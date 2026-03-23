export const DOCUMENTATION_HERO_SNAPSHOT = [
    { label: "Escopo", value: "Catálogo documental relacional" },
    { label: "Stack", value: "Next.js + Go API + PostgreSQL + MinIO" },
    { label: "Processamento", value: "Indexação estrutural com evolução assíncrona" },
    { label: "Meta central", value: "Contexto auditável e reproduzível" },
];

export const DOCUMENTATION_PRINCIPLES = [
    {
        title: "Relational first",
        body: "Collections, categorias, documentos e estruturas tipadas são a base do sistema.",
    },
    {
        title: "Thin HTTP boundary",
        body: "Handlers lidam com validação, auth e serialização; regras ficam em use cases e repositórios.",
    },
    {
        title: "Provider agnostic",
        body: "Storage e LLM podem ser trocados sem reescrever a lógica principal.",
    },
    {
        title: "Academic rigor",
        body: "A implementação precisa refletir com honestidade o que é código atual e o que ainda é roadmap.",
    },
];

export const DOCUMENTATION_ARCHITECTURE_LAYERS = [
    {
        label: "Frontend",
        description: "autenticação, dashboard, preview e documentação.",
    },
    {
        label: "API",
        description: "upload, providers, documentos e recuperação de contexto.",
    },
    {
        label: "Processamento",
        description: "indexação estrutural e enriquecimento opcional.",
    },
    {
        label: "Persistência",
        description: "PostgreSQL e object storage compatível com S3.",
    },
];

export const DOCUMENTATION_ARCHITECTURE_GOALS = [
    "Preservar contexto com joins previsíveis.",
    "Evitar N+1 e montagem quadrática na API.",
    "Permitir portabilidade de providers e ambientes.",
    "Manter consistência entre preview, indexação e contexto consolidado.",
];

export const DOCUMENTATION_ARCHITECTURE_SNAPSHOT = `[Client Workspace]
  -> auth / dashboard / preview / settings

[Go API]
  -> auth
  -> upload + metadata
  -> collection context
  -> provider configuration

[Processing]
  -> outbox dispatch
  -> document parsing
  -> structured indexing

[Persistence]
  -> object storage
  -> relational store`;

export const DOCUMENTATION_PROCESSING_STEPS = [
    {
        step: "01",
        title: "Upload e persistência",
        description: "O documento é validado, armazenado e vinculado a uma collection obrigatória.",
    },
    {
        step: "02",
        title: "Despacho opcional",
        description: "O backend pode registrar eventos em outbox e despachar indexação sem bloquear a interface.",
    },
    {
        step: "03",
        title: "Indexação estrutural",
        description: "Parágrafos, tabelas, outline e metadados são persistidos como estruturas tipadas.",
    },
    {
        step: "04",
        title: "Consulta do contexto",
        description: "O frontend consome o contexto consolidado por collection e documento para navegação e análise.",
    },
];

export const DOCUMENTATION_ROADMAP_ITEMS = [
    "Encerrar os refinamentos visuais menores que restaram no backlog.",
    "Tratar novos ajustes de naming e boilerplate como manutenção normal, não como uma trilha estrutural aberta.",
    "Manter comparações com GraphRAG e abordagens vetoriais como trilhas acadêmicas separadas do produto principal.",
];

export const DOCUMENTATION_REFERENCE_TRACKS = [
    {
        label: "Runtime atual",
        references: ["README.md", "api/README.md", "frontend/README.md", "esta página"],
    },
    {
        label: "Prontidão atual",
        references: ["docs/functional-readiness-spec.md"],
    },
    {
        label: "Pesquisa e roadmap",
        references: ["docs/proposta-continuacao-tcc.md", "docs/roadmap/"],
    },
    {
        label: "Índice das docs secundárias",
        references: ["docs/README.md"],
    },
];
