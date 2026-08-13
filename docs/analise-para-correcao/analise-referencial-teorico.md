# Análise da Seção 4 — Referencial Teórico

## Diagnóstico Geral

A seção atual tem **5 subseções** que somam ~1 página. Seu orientador pediu **2 a 3 páginas** com explicações reais. O problema central é:

> O texto atual **cita autores mas não explica conceitos**. É o padrão "cita e corre" típico de texto gerado por IA.

---

## Problema por Subseção

### 4.1 Engenharia de Software e Pesquisa Aplicada (linhas 131-135)
**Diagnóstico:** Não é referencial teórico. É metodologia. Hevner (DSR) e Pressman (ciclo de vida) são sobre **como fazer pesquisa**, não sobre conceitos fundamentais do domínio.

**Ação:** REMOVER daqui. Mover para a seção de Metodologia.

---

### 4.2 Arquitetura em Camadas e Fronteiras de Domínio (linhas 137-145)
**Diagnóstico:** Não é referencial teórico. É decisão de projeto. Perry & Wolf, Martin (Clean Architecture) e Evans (DDD) são sobre **como projetar o software**, não sobre fundamentos teóricos do problema.

**Ação:** REMOVER daqui. Mover para a seção de Proposta.

---

### 4.3 Modelagem Relacional como Estratégia de Auditabilidade (linhas 147-153)
**Diagnóstico:** É o único tópico que faz sentido aqui, mas é tratado superficialmente.
- Codd (1970) tem **2 frases** — o artigo seminal do modelo relacional merece um parágrafo inteiro
- Buneman (proveniência) tem **2 frases** — o conceito central do seu projeto (auditabilidade) é mencionado de passagem
- Kleppmann (consistência) é fora de contexto — ele fala de sistemas distribuídos, seu sistema é monolítico

**Ação:** REESCREVER. Explicar o que é modelo relacional, normalização, ACID, proveniência.

---

### 4.4 Recuperação de Informação e o Estado da Arte em RAG (linhas 155-161)
**Diagnóstico:** PROBLEMA GRAVE. RAG é o conceito central do seu projeto, mas tem **2 frases**.
- Lewis et al. (2020) — o artigo que INVENTOU o RAG — é mencionado em meia frase
- "Documentos são convertidos em vetores e consultados por similaridade de cosseno" — não explica o pipeline, não explica por que isso é relevante
- Xu (LayoutLM) é citado sem contexto

**Ação:** REESCREVER. Gastar 1 página inteira explicando RAG: o problema que resolve, o pipeline, as limitações, como ThingIndex se posiciona.

---

### 4.5 Trabalhos Relacionados (linhas 163-191)
**Diagnóstico:** Deveria ser seção separada. Mistura comparação técnica com análise de concorrentes. Não menciona Notion, Outline, PageIndex (seu orientador pediu).

**Ação:** REMOVER daqui. Virar Seção 3 "Trabalhos Relacionados".

---

## Nova Estrutura Proposta para Seção 4

| Subseção | Conteúdo | Autores-chave | Tamanho |
|----------|----------|---------------|---------|
| 4.1 | Inteligência Documental e Gestão do Conhecimento | Nonaka & Takeuchi, Davenport & Prusak | 25-30 linhas |
| 4.2 | Recuperação de Informação e Indexação Semântica | Salton, Baeza-Yates, Manning, Mikolov, Devlin | 30-35 linhas |
| 4.3 | RAG — Retrieval-Augmented Generation | Lewis et al. (2020) | 35-40 linhas |
| 4.4 | Modelagem Relacional para Sistemas Documentais | Codd (1970), Date, Elmasri, Buneman | 30-35 linhas |
| 4.5 | Panorama Tecnológico | — (justificativas) | 20-25 linhas |

**Total estimado: 140-170 linhas = ~2.5 a 3 páginas** ✅

---

## Regras de Ouro para a Nova Versão

1. **Toda citação deve vir com explicação.** Não escreva "Codd propôs o modelo relacional". Escreva "Codd propôs o modelo relacional, que organiza dados em tabelas (relações) com linhas (tuplas) e colunas (atributos), garantindo que cada registro seja único por meio de chaves primárias..."

2. **Conecte cada conceito ao ThingIndex.** No final de cada subseção, explique como aquele conceito se aplica ao seu projeto.

3. **Escreva com suas palavras.** Se parece que saiu do ChatGPT, seu orientador vai perceber. Use frases mais curtas. Explique como se estivesse ensinando um colega.

4. **Não comece com frases genéricas.** Evite: "O crescimento do volume de documentos digitais..." Comece direto no conceito.

5. **Gaste mais tempo em RAG (4.3).** É o conceito mais importante para seu projeto. É onde você mostra que entende do assunto.
