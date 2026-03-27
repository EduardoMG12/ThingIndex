# RFC 0001 - Foco do Produto ThingIndex

**Status:** aceito  
**Data:** 2026-03-22  
**Responsável:** Charles Eduardo Mello Guimaraes  
**Escopo:** posicionamento do produto, recorte do runtime principal, impacto no backlog e na escrita do TCC

## 1. Objetivo deste RFC

Este RFC existe para resolver um problema específico do projeto:

> o repositório já possui implementação suficiente para demonstrar um produto funcional, mas a narrativa ainda corre o risco de misturar runtime atual, extensão acadêmica e experimento futuro.

Em outras palavras, o objetivo deste documento não é "fazer marketing". O objetivo é:

- definir com precisão o que o ThingIndex é hoje;
- explicitar para quem ele serve;
- registrar o que não deve ser prometido como funcionalidade pronta;
- alinhar backlog, documentação pública e artigo do TCC ao mesmo recorte.

## 2. Contexto

O estado atual do repositório já entrega:

- frontend em Next.js;
- backend HTTP em Go;
- PostgreSQL como base relacional principal;
- MinIO / S3-compatible para armazenamento;
- RabbitMQ para indexação assíncrona;
- indexação estrutural com persistência de metadados, parágrafos e tabelas;
- recuperação de contexto por `collection`;
- chat grounded por `collection`;
- documentação pública via Scalar e Swagger.

O principal risco atual deixou de ser técnico-operacional. O risco mais relevante passou a ser **narrativo**:

- apresentar GraphRAG, vetores ou assistentes mais ambiciosos como se já fossem parte do caminho principal;
- deixar o produto parecer genérico demais;
- enfraquecer a defesa do TCC por falta de um recorte claro.

## 3. Problema

Sem um foco explícito, o projeto passa mensagens conflitantes:

- para o usuário, não fica claro qual problema ele resolve melhor;
- para o colaborador, o backlog parece puxar para direções demais;
- para o TCC, a contribuição fica difusa;
- para documentação e landing page, a copy tende a prometer mais do que o runtime realmente comprova.

## 4. Decisão

O ThingIndex passa a ser definido como:

> **uma plataforma relacional de inteligência documental para workspaces auditáveis**, voltada a coleções de documentos que exigem organização, extração estruturada, recuperação de contexto e rastreabilidade operacional.

### 4.1 Caso de uso principal

O caso de uso principal do runtime atual é:

- organizar coleções documentais operacionais, jurídicas, acadêmicas ou de compliance;
- preservar contexto e proveniência por modelagem relacional explícita;
- permitir navegação, análise e chat grounded sobre esse contexto já consolidado.

### 4.2 Público-alvo prioritário

O foco primário do produto hoje são:

- times de operações;
- times de compliance e governança;
- fluxos jurídicos ou normativos;
- contextos acadêmicos em que rastreabilidade e clareza arquitetural importam.

## 5. O que o produto promete hoje

O runtime principal assume como promessa:

- organização documental por categorias, `collections` e documentos;
- recuperação de contexto relacional previsível;
- extração e persistência de estruturas tipadas;
- observabilidade e auditoria operacional;
- portabilidade de providers sem reescrita do domínio principal.

## 6. O que o produto não promete hoje

As seguintes leituras estão explicitamente fora do escopo principal:

- "copiloto geral de IA" desvinculado de documentos e proveniência;
- produto centrado em banco de grafos;
- produto vetor-first;
- assistente jurídico pronto para produção;
- equivalência de maturidade entre todos os providers LLM.

## 7. Alternativas consideradas

### 7.1 Manter o projeto genérico como "plataforma de IA documental"

Rejeitada porque:

- dilui o diferencial relacional;
- enfraquece a defesa técnica;
- aumenta risco de promessas ambíguas.

### 7.2 Reposicionar tudo como projeto GraphRAG / vetorial

Rejeitada porque:

- não representa o runtime implementado;
- desloca o grupo de controle relacional já existente;
- transforma pesquisa futura em marketing presente.

### 7.3 Tratar o repositório apenas como base acadêmica, não como produto

Rejeitada porque:

- ignora o valor prático já implementado;
- enfraquece a frente open source;
- reduz a utilidade do próprio sistema para demonstração e continuidade.

## 8. Implicações práticas

Esta decisão implica que:

- o app deve expor publicamente o foco do runtime;
- landing, `/documentation`, README e API docs devem refletir o mesmo recorte;
- o artigo do TCC deve tratar o runtime atual como baseline implementada;
- vetores e GraphRAG devem ser descritos como trilhas comparativas, não como núcleo do produto.

## 9. Critérios de aceite

Este RFC é considerado efetivamente aplicado quando:

- existir uma definição pública do foco do runtime;
- a documentação pública repetir esse foco sem contradição;
- o backlog for reorganizado a partir dele;
- o artigo derivado adotar o mesmo recorte.

## 10. Impacto no backlog

Depois desta decisão, a ordem lógica muda:

1. avaliar acurácia e groundedness de forma objetiva;
2. melhorar qualidade percebida do chat e do monitoramento;
3. expandir storages e CLI apenas depois do núcleo de valor estar melhor medido.

## 11. Artefatos vinculados

- `docs/article/thingindex-sbc-article.md`
- `docs/article/sbc_template (1).doc`
- `docs/proposta-continuacao-tcc.md`
- `what-missing?.md`
