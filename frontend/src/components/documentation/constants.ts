export const TOC = [
  { id: "overview", label: "Visão Geral" },
  { id: "principles", label: "Princípios de Projeto" },
  { id: "architecture", label: "Arquitetura do Sistema" },
  { id: "pipeline", label: "Pipeline de Processamento" },
  { id: "data-model", label: "Modelo de Dados" },
  { id: "retrieval", label: "Retrieval e Indexação" },
  { id: "security", label: "Segurança e Governança" },
  { id: "deployment", label: "Deploy e Operação" },
  { id: "providers", label: "Providers e Portabilidade" },
  { id: "pricing", label: "Custos e Benchmark" },
  { id: "academic", label: "TCC e Artigo" },
  { id: "contributing", label: "Contribuição e Roadmap" },
  { id: "references", label: "Referências e Materiais" },
];

export const FEATURES = [
  { feature: "Chunks por documento", rag: "Fixo (512/1024 tokens)", thingindex: "Adaptativo por página + estrutura" },
  { feature: "Metadados por chunk", rag: "Mínimo (doc_id)", thingindex: "Ricos: page_index, titulo, subtitulo, assunto, nodes, categoria, collection" },
  { feature: "Cadeia de prompts", rag: "1 prompt (embed)", thingindex: "3 prompts: Resumo → Estruturação → Embedding" },
  { feature: "Organização", rag: "Flat (1 index)", thingindex: "Hierárquica: Categoria → Collection → Documento → Página" },
  { feature: "Grafo de conhecimento", rag: "Não", thingindex: "Sim — entidades extraídas como nós interconectados" },
  { feature: "Busca", rag: "Similaridade vetorial pura", thingindex: "Vetorial + filtro por categoria/collection + grafo" },
  { feature: "Rastreabilidade", rag: "Limitada", thingindex: "Total: doc → página → chunk → prompt usado" },
  { feature: "Validação de segurança", rag: "Opcional", thingindex: "Obrigatório: ClamAV scan antes de processar" },
];

export const MODELS = [
  { name: "Gemini 2.5 Pro", provider: "Google", input: "$1.25", output: "$5.00", context: "1M tokens", notes: "Melhor custo-benefício para análise documental" },
  { name: "Gemini 2.5 Flash", provider: "Google", input: "$0.15", output: "$0.60", context: "1M tokens", notes: "Rápido e econômico para tarefas simples" },
  { name: "GPT-4o", provider: "OpenAI", input: "$2.50", output: "$10.00", context: "128K tokens", notes: "Alta qualidade, custo mais elevado" },
  { name: "GPT-4o Mini", provider: "OpenAI", input: "$0.15", output: "$0.60", context: "128K tokens", notes: "Alternativa econômica da OpenAI" },
  { name: "Claude 3.5 Sonnet", provider: "Anthropic", input: "$3.00", output: "$15.00", context: "200K tokens", notes: "Excelente raciocínio, custo premium" },
  { name: "Claude 3.5 Haiku", provider: "Anthropic", input: "$0.80", output: "$4.00", context: "200K tokens", notes: "Boa relação qualidade/preço" },
  { name: "DeepSeek V3", provider: "DeepSeek", input: "$0.27", output: "$1.10", context: "128K tokens", notes: "Open-source, custo muito baixo" },
  { name: "Llama 3.1 405B", provider: "Meta", input: "$0.00", output: "$0.00", context: "128K tokens", notes: "Gratuito (self-hosted), requer GPU" },
];

export const EMBEDDINGS = [
  { name: "text-embedding-3-small", provider: "OpenAI", price: "$0.02/1M tokens", dims: "1536", notes: "Padrão ThingIndex" },
  { name: "text-embedding-3-large", provider: "OpenAI", price: "$0.13/1M tokens", dims: "3072", notes: "Maior precisão" },
  { name: "Gecko", provider: "Google", price: "$0.025/1M tokens", dims: "768", notes: "Integrado ao Vertex AI" },
  { name: "BGE-M3", provider: "BAAI", price: "Gratuito", dims: "1024", notes: "Open-source, self-hosted" },
];
