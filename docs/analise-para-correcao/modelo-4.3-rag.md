# Modelo de Texto — 4.3 RAG (Retrieval-Augmented Generation)

## IMPORTANTE

Esta é a subseção mais importante do Referencial Teórico. Seu orientador EXPLICITAMENTE pediu para explicar RAG "inventado por x e é assim y". Gaste tempo aqui.

---

## Texto Modelo

### 4.3 RAG — Retrieval-Augmented Generation

Modelos de linguagem de grande escala (LLMs), como o GPT (OpenAI), LLaMA (Meta) e Gemini (Google), demonstraram capacidade impressionante de gerar texto coerente e responder perguntas sobre uma ampla gama de assuntos. No entanto, esses modelos têm uma limitação fundamental: seu conhecimento é limitado aos dados com os quais foram treinados, que podem estar desatualizados ou não cobrir domínios específicos. Quando confrontados com perguntas sobre documentos que não fazem parte de seu treinamento, LLMs tendem a **alucinar** — isto é, gerar respostas confiantes, porém factualmente incorretas (Ji et al., 2023).

**RAG** (Retrieval-Augmented Generation) é uma arquitetura proposta por Lewis et al. (2020), pesquisadores do Facebook AI, no artigo "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks". A ideia central é combinar um sistema de recuperação de informação com um modelo gerativo:

1. o **retriever** busca documentos ou trechos relevantes em uma base de conhecimento externa;
2. o **generator** (um LLM) recebe esses trechos como contexto adicional junto com a pergunta do usuário e gera uma resposta fundamentada.

**Pipeline típico do RAG.** O fluxo de operação de um sistema RAG pode ser descrito em cinco etapas:

1. **Ingestão:** o documento original é recebido pelo sistema.
2. **Fragmentação (chunking):** o documento é dividido em pedaços menores (chunks), geralmente de 256 a 1024 tokens, com ou sem sobreposição. O tamanho e a estratégia de chunking impactam diretamente a qualidade da recuperação.
3. **Indexação vetorial:** cada chunk é convertido em um vetor denso (embedding) por meio de um modelo como text-embedding-3-small (OpenAI) ou BGE (BAAI). Esses vetores são armazenados em um **banco de dados vetorial** (vector DB), como Pinecone, Weaviate, Milvus ou pgvector.
4. **Consulta:** a pergunta do usuário é convertida no mesmo espaço vetorial. O vector DB retorna os K chunks mais similares (por similaridade de cosseno ou distância euclidiana).
5. **Geração:** os chunks recuperados são concatenados com a pergunta original e enviados ao LLM, que gera uma resposta com base no contexto fornecido.

**Vantagens do RAG.** A principal vantagem é que o conhecimento pode ser atualizado sem necessidade de retreinar o modelo — basta substituir ou adicionar documentos na base. Além disso, o RAG permite que o LLM acesse informações específicas de um domínio ou organização sem que essas informações precisem estar em seu treinamento original.

**Limitações do RAG tradicional.** Apesar de sua popularidade, o RAG convencional apresenta problemas significativos de **opacidade**:

- A fragmentação (chunking) é geralmente feita de forma cega, baseada em contagem de tokens, sem respeitar a estrutura semântica do documento. Um parágrafo pode ser cortado ao meio; uma tabela pode ser separada de sua legenda.
- A recuperação por similaridade vetorial é uma **caixa-preta**: o sistema retorna chunks que "são similares" à consulta, mas não explica por que foram considerados relevantes, nem permite que o usuário inspecione o critério de matching.
- A **auditabilidade** é limitada: mesmo que o LLM cite fontes, rastrear de que documento, página e parágrafo veio cada informação exige mecanismos adicionais que a maioria das implementações não oferece.
- O custo operacional depende de serviços externos (API de embeddings + vector DB + LLM), o que torna o custo variável e imprevisível.

**Relação com o ThingIndex.** O ThingIndex se propõe a explorar uma alternativa à arquitetura RAG tradicional. Em vez de fragmentar documentos em chunks genéricos e indexá-los por similaridade vetorial, o sistema preserva a estrutura semântica original (parágrafos, tabelas, metadados) em tabelas relacionais tipadas. A recuperação de contexto ocorre por meio de consultas SQL determinísticas sobre chaves estrangeiras, não por similaridade de cosseno em um espaço vetorial.

Essa abordagem não elimina o uso de LLMs — o ThingIndex prevê integração opcional com provedores de linguagem para enriquecimento e chat — mas substitui o vector DB por uma **baseline relacional auditável**. O objetivo é demonstrar que, para cenários onde a rastreabilidade da informação é crítica (como processos acadêmicos, jurídicos ou de compliance), uma abordagem relacional rigorosa pode oferecer vantagens de auditabilidade e previsibilidade de custo que o RAG tradicional não oferece.

---

## Fontes para consultar no Google Acadêmico

- Lewis, P., et al. (2020). "Retrieval-Augmented Generation for Knowledge-Intensive NLP Tasks". *NeurIPS*.
- Ji, Z., et al. (2023). "Survey of Hallucination in Natural Language Generation". *ACM Computing Surveys*.
- Gao, Y., et al. (2023). "Retrieval-Augmented Generation for Large Language Models: A Survey". *arXiv*.

---

## Checklist

- [ ] Li o artigo original do RAG (Lewis et al., 2020)
- [ ] Sei explicar o pipeline de 5 etapas sem ler
- [ ] Entendo por que chunking cego é um problema
- [ ] Consigo explicar a diferença entre vector DB e tabelas relacionais
- [ ] Conectei RAG ao ThingIndex (substituição do vector DB por SQL)
- [ ] O texto tem CITAÇÕES REAIS que eu verifiquei
