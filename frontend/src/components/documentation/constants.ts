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
    { feature: "Arquitetura", rag: "Chunk-first ou vector-first", thingindex: "Camadas Go/Next.js com dominio relacional" },
    { feature: "Recuperacao principal", rag: "Similaridade semantica sobre chunks", thingindex: "Contexto consolidado por collection e estruturas tipadas" },
    { feature: "Rastreabilidade", rag: "Normalmente centrada no chunk", thingindex: "Documento, collection, metadados e tentativas operacionais" }
];

export const MODELS: DocumentationModel[] = [
    {
        name: "OpenAI-compatible chat",
        provider: "OpenAI-compatible",
        input: "Variavel",
        output: "Variavel",
        context: "Provider-defined",
        notes: "Os custos reais dependem da configuracao do usuario e da sincronizacao de pricing.",
    },
    {
        name: "Ollama local chat",
        provider: "Ollama",
        input: "Infra local",
        output: "Infra local",
        context: "Modelo local",
        notes: "Adequado para ambiente self-hosted; custo nao deve ser tratado como tabela fixa do runtime.",
    }
];

export const EMBEDDINGS: DocumentationEmbedding[] = [
    {
        name: "Trilha vetorial experimental",
        provider: "Pesquisa",
        price: "Fora do mainline",
        dims: "N/A",
        notes: "Comparacoes vetoriais pertencem ao roadmap academico e nao a navegacao principal atual.",
    },
    {
        name: "Precificacao sincronizada",
        provider: "Runtime",
        price: "Dinamica",
        dims: "N/A",
        notes: "A verdade operacional sobre custo deve vir de precos sincronizados e uso observado.",
    }
];
