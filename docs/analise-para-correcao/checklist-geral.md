# Checklist Geral — Correção do Artigo ThingIndex (TCC1)

## Seção 4 — Referencial Teórico

### 4.1 Inteligência Documental e Gestão do Conhecimento
- [ ] Definição de conhecimento tácito vs explícito (Nonaka & Takeuchi)
- [ ] Espiral do conhecimento explicada
- [ ] Hierarquia DIKW (dado → informação → conhecimento)
- [ ] Conexão com ThingIndex ao final

### 4.2 Recuperação de Informação e Indexação Semântica
- [ ] O que é RI (Manning et al.)
- [ ] TF-IDF explicado (como funciona o cálculo)
- [ ] BM25 explicado (Robertson & Zaragoza)
- [ ] Limitação do matching léxico
- [ ] Embeddings e Word2Vec (Mikolov)
- [ ] BERT e embeddings contextuais (Devlin)
- [ ] Similaridade de cosseno
- [ ] Conexão com ThingIndex (joins SQL vs similaridade vetorial)

### 4.3 RAG — Retrieval-Augmented Generation
- [ ] Problema que RAG resolve (alucinação em LLMs)
- [ ] Lewis et al. (2020) — quem inventou e quando
- [ ] Pipeline completo explicado: chunking → embedding → vector DB → consulta → LLM
- [ ] Vantagens do RAG
- [ ] Limitações: chunking cego, caixa-preta da recuperação, auditabilidade
- [ ] Conexão com ThingIndex (substituição do vector DB por SQL)

### 4.4 Modelagem Relacional para Sistemas Documentais
- [ ] Codd (1970) — o que é modelo relacional
- [ ] Chave primária, chave estrangeira, integridade referencial
- [ ] Normalização (1FN, 2FN, 3FN)
- [ ] Propriedades ACID
- [ ] Proveniência (Buneman et al.)
- [ ] Conexão com ThingIndex (20 tabelas, joins SQL, auditabilidade)

### 4.5 Panorama Tecnológico
- [ ] Next.js — por que foi escolhido (comparação com alternativas)
- [ ] Go — por que foi escolhido (goroutines, concorrência)
- [ ] PostgreSQL — por que relacional (ACID, consistência)
- [ ] Object Storage — por que separado do banco
- [ ] RabbitMQ — por que filas (outbox, retry, DLQ)
- [ ] Docker Compose — reprodutibilidade
- [ ] SEM versões específicas (Go 1.25, Next.js 16)

---

## Seção 3 — Trabalhos Relacionados (NOVA)
- [ ] Notion — quem criou, como funciona, limitação
- [ ] Outline — quem criou, como funciona, limitação
- [ ] LangChain — quem criou, como funciona, limitação
- [ ] LlamaIndex — quem criou, como funciona, limitação
- [ ] Tabela comparativa com critérios objetivos
- [ ] Parágrafo de lacuna (o que nenhum deles oferece)

---

## Seção 5 — Proposta (NOVA)
- [ ] 5.1 O que é o ThingIndex (definição, problema, público)
- [ ] 5.2 Arquitetura (diagrama em camadas)
- [ ] 5.3 Modelo de dados (ER simplificado)
- [ ] 5.4 Protótipo das telas (imagens)
- [ ] 5.5 Fluxo de processamento (fluxograma: upload → fila → indexação → consulta)
- [ ] 5.6 Casos de uso (diagrama UML)
- [ ] 5.7 Avaliação (como medir se é bom ou ruim)
- [ ] 5.8 Cronograma (junho até defesa)
- [ ] NENHUMA linha de código

---

## Estilo e Qualidade
- [ ] Artigo escrito no PRESENTE (não futuro do pretérito)
- [ ] NENHUM auto-elogio ("se distingue", "postura menos comum")
- [ ] Frases curtas e diretas
- [ ] Conceitos explicados, não apenas citados
- [ ] Não começa com frases genéricas ("crescimento do volume...")
- [ ] Nota sobre uso de IA (correção ortográfica apenas)
- [ ] PDF apenas (remover HTML, XLSX)

---

## Referências
- [ ] Todas as citações existem no arquivo .bib
- [ ] Autores e anos conferidos
- [ ] URLs funcionam (se houver)
- [ ] Formato SBC (template da revista)
