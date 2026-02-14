# LegisGraph: Knowledge Graph for Brazilian Legislation (TCC)

This project is a **Course Completion Work (TCC)** aimed at building a structured knowledge graph of Brazilian federal legislation (Planalto.gov.br).

## 🎯 Objective

The central idea is to **create a rich data structure to improve AI responses (RAG)** compared to traditional vector-only approaches.

### The Problem with Standard RAG
Traditional RAG (Retrieval-Augmented Generation) uses vector similarity to find relevant text chunks. However, legal reasoning relies heavily on **hierarchical context** and **relationships**:
- "This law was revoked by X"
- "Article 5 applies, except as modified by Law Y"
- "This paragraph defines the term used in the Inciso above"

Vector search often loses this structural context, leading to hallucinations or missing critical cross-references.

### The Solution: Graph RAG
LegisGraph parses laws into a graph structure where:
- **Nodes** are laws, articles, paragraphs.
- **Edges** represent hierarchy (*contains*) and relationships (*revokes*, *alters*, *references*).

This allows an AI agent to traverse the graph and retrieve the **exact legal context** (e.g., "The specific valid version of Art. 5 along with the specific law that amended it"), providing far more accurate answers.

## 🏗️ Architecture

The system consists of three main components:

1. **[Crawler](./crawler)**:
   - Traverses Planalto.gov.br.
   - Captures hierarchical context of links (e.g., "Link found in Art. 2, § 3").
   - Saves raw HTML/PDF and comprehensive metadata.

2. **[Legal-Parser](./legal-parser)**:
   - Processes raw documents into structured JSON.
   - Extracts hierarchy (Art., §, Inciso, Alínea).
   - Enriches cross-references with their origin context.
   - Outputs standardized JSON ready for graph ingestion.

3. **Knowledge Graph (Neo4j)**:
   - Ingests the parsed JSON.
   - Models the complex web of Brazilian legislation.

## 🚀 Key Features

- **Hierarchical Context Tracking**: Knows exactly where a reference came from (e.g., "Link to Law Y found inside Art. X of Constitution").
- **Canonical IDs**: Unique, idempotent identifiers for every legal document (e.g., `lei-federal-8078-1990`).
- **Rich Metadata**: Captures provenance, dates, and structural lineage.

---
*Developed as part of a Computer Science TCC project.*
