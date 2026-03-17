export const TOC = [
  { id: "overview", label: "Visão Geral" },
  { id: "principles", label: "Princípios de Projeto" },
  { id: "architecture", label: "Arquitetura do Sistema" },
  { id: "processing", label: "Processamento Atual" },
  { id: "data-model", label: "Modelo de Dados" },
  { id: "retrieval", label: "Recuperação de Contexto" },
  { id: "security", label: "Segurança e Governança" },
  { id: "deployment", label: "Deploy e Operação" },
  { id: "providers", label: "Providers e Portabilidade" },
  { id: "roadmap", label: "Roadmap" },
  { id: "academic", label: "TCC e Artigo" },
  { id: "contributing", label: "Contribuição" },
  { id: "references", label: "Referências e Materiais" },
];

export type DocumentationFeature = {
    feature: string;
    rag: string;
    thingindex: string;
};

export type DocumentationModel = {
    name: string;
    provider: string;
    input: string;
    output: string;
    context: string;
    notes: string;
};

export type DocumentationEmbedding = {
    name: string;
    provider: string;
    price: string;
    dims: string;
    notes: string;
};

export const FEATURES: DocumentationFeature[] = [
    { feature: "Arquitetura", rag: "Monolítica", thingindex: "Microsserviços Event-Driven" },
    { feature: "Busca", rag: "Apenas Vetorial", thingindex: "Híbrida (Grafo de Conhecimento + Vetores)" },
    { feature: "Rastreabilidade", rag: "Limitada", thingindex: "Total (Página, Parágrafo, Metadados)" }
];

export const MODELS: DocumentationModel[] = [
    { name: "gpt-4-turbo", provider: "OpenAI", input: "$10.00", output: "$30.00", context: "128k", notes: "Alta qualidade analítica" },
    { name: "gpt-3.5-turbo", provider: "OpenAI", input: "$0.50", output: "$1.50", context: "16k", notes: "Rápido e barato" }
];

export const EMBEDDINGS: DocumentationEmbedding[] = [
    { name: "text-embedding-3-small", provider: "OpenAI", price: "$0.02", dims: "1536", notes: "Otimizado para custo" },
    { name: "text-embedding-3-large", provider: "OpenAI", price: "$0.13", dims: "3072", notes: "Alta precisão semântica" }
];
