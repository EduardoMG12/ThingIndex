# Modelo de Texto — 4.2 Recuperação de Informação e Indexação Semântica

## Texto Modelo

### 4.2 Recuperação de Informação e Indexação Semântica

Recuperação de Informação (RI) é a área da ciência da computação que estuda como encontrar documentos relevantes a partir de uma necessidade de informação expressa por um usuário (Manning et al., 2008). Diferente de uma consulta a um banco de dados tradicional, onde a estrutura é conhecida e os dados são precisos, a RI lida com texto não estruturado e relevância aproximada.

**Métodos Clássicos.** O modelo mais influente de RI é o **TF-IDF** (Term Frequency — Inverse Document Frequency), proposto por Salton (1989). O TF-IDF calcula um peso para cada termo em cada documento: a frequência do termo no documento (TF) multiplicada pelo inverso da frequência do termo no corpus (IDF). Termos que aparecem muitas vezes em um documento, mas são raros no corpus geral, recebem peso alto — são os termos que melhor caracterizam aquele documento.

Robertson e Zaragoza (2009) aperfeiçoaram essa abordagem com o **BM25** (Best Matching 25), um modelo probabilístico que incorpora ajustes como saturação de frequência de termo e normalização por tamanho do documento. O BM25 é até hoje o padrão da indústria para busca textual em sistemas como Elasticsearch e Lucene.

A limitação central dos métodos clássicos é o **matching léxico**: eles só encontram correspondência exata entre os termos da consulta e os termos do documento. Uma busca por "automóvel" não encontra documentos que mencionam "carro", a menos que ambos os termos estejam indexados.

**Métodos Neurais.** A partir de 2013, com a popularização dos **word embeddings** proposta por Mikolov et al. (2013) no Word2Vec, tornou-se possível representar palavras como vetores densos em um espaço contínuo de centenas de dimensões. Palavras com significado semelhante (como "carro" e "automóvel") tendem a ocupar posições próximas nesse espaço vetorial.

A evolução seguinte foram os **embeddings contextuais**, como o BERT (Devlin et al., 2019). Diferente do Word2Vec, que gera um vetor fixo para cada palavra, o BERT gera vetores que variam conforme o contexto — a palavra "banco" tem um vetor diferente em "banco de dados" e em "banco do parque". Isso permitiu avanços significativos em tarefas de busca semântica.

**Indexação Semântica.** A combinação de embeddings com mecanismos de similaridade (especialmente **similaridade de cosseno**) deu origem à indexação semântica: documentos e consultas são convertidos para vetores, e a busca ocorre por proximidade vetorial em vez de correspondência de termos. Essa abordagem é a base de sistemas modernos de busca e de arquiteturas como o RAG.

No ThingIndex, a recuperação de informação adota uma abordagem diferente. Em vez de converter documentos em vetores, o sistema preserva a estrutura textual original em tabelas tipadas (parágrafos, seções, tabelas) e recupera o contexto por meio de junções SQL (JOINs) sobre chaves estrangeiras. Essa escolha não busca competir com a flexibilidade semântica dos métodos neurais, mas oferece **determinismo e auditabilidade**: cada trecho recuperado tem uma cadeia de rastreamento que leva ao documento, página e parágrafo de origem. O referencial de RI clássica é mantido como baseline sobre a qual extensões futuras (como embeddings opcionais) poderão ser comparadas.

---

## Checklist

- [ ] Entendo a diferença entre TF-IDF e BM25
- [ ] Sei explicar por que matching léxico é limitado
- [ ] Entendo o que é um embedding e similaridade de cosseno
- [ ] Conectei o conceito ao ThingIndex (joins SQL vs similaridade vetorial)
- [ ] O texto não parece cópia da Wikipedia
