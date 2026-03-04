"""
Centralized repository for Neo4j Cypher queries.
Keeps the database client decoupled from specific business logic queries.
"""

# Query to get the main topics connected to the origin node.
GET_TOPICS_QUERY = """
MATCH (t:Topic)-[:PERTENCE]->(o:Origem {name: "Brasil"})
RETURN t.name as topic
ORDER BY t.name
"""

# ---------- ORIGINAL QUERY (Legacy 'all' fallback) ----------
# Note: This query uses Python string formatting for the LIMIT clause
GET_VISUALIZE_GRAPH_QUERY = """
// Priority sorting to ensure cohesive expansion from the root
MATCH (n)-[r]->(m)
WITH n, r, m,
     CASE 
       WHEN 'Origem' IN labels(n) OR 'Origem' IN labels(m) THEN 0
       WHEN 'Topic' IN labels(n) OR 'Topic' IN labels(m) THEN 1
       WHEN ('Lei' IN labels(n) OR 'LEI' IN labels(n) OR 'Lei' IN labels(m) OR 'LEI' IN labels(m)) THEN 2
       WHEN 'Artigo' IN labels(n) OR 'Artigo' IN labels(m) THEN 3
       WHEN 'Paragrafo' IN labels(n) OR 'Paragrafo' IN labels(m) THEN 4
       WHEN 'Inciso' IN labels(n) OR 'Inciso' IN labels(m) THEN 5
       ELSE 6
     END as priority
ORDER BY priority ASC
LIMIT {limit}

// Ensure nodes are unique
WITH collect(distinct n) + collect(distinct m) as all_nodes, collect(r) as all_rels

RETURN 
    [node IN all_nodes | {{
        id: toString(elementId(node)), 
        label: CASE
                  WHEN node.numero IS NOT NULL THEN
                    CASE 
                      WHEN 'Artigo' IN labels(node) THEN "Art. " + node.numero
                      WHEN 'Paragrafo' IN labels(node) THEN "§ " + node.numero
                      WHEN 'Inciso' IN labels(node) THEN "Inc. " + node.numero
                      WHEN 'Alinea' IN labels(node) THEN "Al. " + node.numero
                      ELSE node.numero
                    END
                  ELSE COALESCE(node.name, node.id, split(labels(node)[0], ':')[0])
               END,
        group: CASE 
                 WHEN 'Origem' IN labels(node) THEN 0
                 WHEN 'Topic' IN labels(node) THEN 1
                 WHEN 'Lei' IN labels(node) OR 'LEI' IN labels(node) OR 'Decreto' IN labels(node) OR 'DECRETO' IN labels(node) OR 'MPV' IN labels(node) OR 'LC' IN labels(node) OR 'Documento' IN labels(node) OR 'LegalDocument' IN labels(node) THEN 2
                 WHEN 'Artigo' IN labels(node) THEN 3
                 WHEN 'Paragrafo' IN labels(node) THEN 4
                 WHEN 'Inciso' IN labels(node) THEN 5
                 WHEN 'Alinea' IN labels(node) THEN 6
                 ELSE 7 
               END
    }}] as nodes,
    [rel IN all_rels | {{
        source: toString(elementId(startNode(rel))),
        target: toString(elementId(endNode(rel))),
        label: type(rel)
    }}] as links
"""

# ---------- CONTEXT-AWARE QUERIES ----------

# Get total node count in the entire DB
GET_TOTAL_NODES_ALL = """
MATCH (n) RETURN count(n) as total
"""

# Get total node count for a specific topic context
# A node belongs to a topic if it has the topic property set to it, or if it's the topic node itself
GET_TOTAL_NODES_CONTEXT = """
MATCH (n)
WHERE n.topic = $context OR (n:Topic AND n.name = $context)
RETURN count(n) as total
"""

# Fetch the graph but limited strictly to nodes that belong to a specific topic context
# We use neo4j parameters ($context) for the dynamic part, and python format ({limit}) for the limit
GET_VISUALIZE_CONTEXT_QUERY = """
// Priority sorting within context
MATCH (n)-[r]->(m)
WHERE (n.topic = $context OR (n:Topic AND n.name = $context) OR (n:Origem AND n.name="Brasil"))
  AND (m.topic = $context OR (m:Topic AND m.name = $context) OR (m:Origem AND m.name="Brasil"))
WITH n, r, m,
     CASE 
       WHEN 'Origem' IN labels(n) OR 'Origem' IN labels(m) THEN 0
       WHEN 'Topic' IN labels(n) OR 'Topic' IN labels(m) THEN 1
       WHEN ('Lei' IN labels(n) OR 'LEI' IN labels(n) OR 'Lei' IN labels(m) OR 'LEI' IN labels(m)) THEN 2
       WHEN 'Artigo' IN labels(n) OR 'Artigo' IN labels(m) THEN 3
       WHEN 'Paragrafo' IN labels(n) OR 'Paragrafo' IN labels(m) THEN 4
       WHEN 'Inciso' IN labels(n) OR 'Inciso' IN labels(m) THEN 5
       ELSE 6
     END as priority
ORDER BY priority ASC
LIMIT {limit}

WITH collect(distinct n) + collect(distinct m) as all_nodes, collect(r) as all_rels

RETURN 
    [node IN all_nodes | {{
        id: toString(elementId(node)), 
        label: CASE
                  WHEN node.numero IS NOT NULL THEN
                    CASE 
                      WHEN 'Artigo' IN labels(node) THEN "Art. " + node.numero
                      WHEN 'Paragrafo' IN labels(node) THEN "§ " + node.numero
                      WHEN 'Inciso' IN labels(node) THEN "Inc. " + node.numero
                      WHEN 'Alinea' IN labels(node) THEN "Al. " + node.numero
                      ELSE node.numero
                    END
                  ELSE COALESCE(node.name, node.id, split(labels(node)[0], ':')[0])
               END,
        group: CASE 
                 WHEN 'Origem' IN labels(node) THEN 0
                 WHEN 'Topic' IN labels(node) THEN 1
                 WHEN 'Lei' IN labels(node) OR 'LEI' IN labels(node) OR 'Decreto' IN labels(node) OR 'DECRETO' IN labels(node) OR 'MPV' IN labels(node) OR 'LC' IN labels(node) OR 'Documento' IN labels(node) OR 'LegalDocument' IN labels(node) THEN 2
                 WHEN 'Artigo' IN labels(node) THEN 3
                 WHEN 'Paragrafo' IN labels(node) THEN 4
                 WHEN 'Inciso' IN labels(node) THEN 5
                 WHEN 'Alinea' IN labels(node) THEN 6
                 ELSE 7 
               END
    }}] as nodes,
    [rel IN all_rels | {{
        source: toString(elementId(startNode(rel))),
        target: toString(elementId(endNode(rel))),
        label: type(rel)
    }}] as links
"""

# ---------- LAW SEARCH (for AI Assistant) ----------

# Search for nodes by fuzzy matching on name, topic, or numero.
# Returns the full tree: the matched node + ancestors (up to Origin) + descendants (down to Alinea).
# $terms is a list of search strings passed from the LLM's identification step.
SEARCH_LAW_TREE_QUERY = """
UNWIND $terms AS term
OPTIONAL MATCH (n)
WHERE (n.name IS NOT NULL AND toLower(n.name) CONTAINS toLower(term))
   OR (n.numero IS NOT NULL AND toLower(n.numero) CONTAINS toLower(term))
   OR (n.topic IS NOT NULL AND toLower(n.topic) CONTAINS toLower(term))
WITH collect(DISTINCT n) AS matched

// Expand downward from matched nodes (children, grandchildren, etc.)
UNWIND matched AS m
OPTIONAL MATCH path_down = (m)-[*0..4]->(descendant)
WITH matched, collect(DISTINCT descendant) AS descendants

// Expand upward from matched nodes (parents, grandparents to Origin)
UNWIND matched AS m
OPTIONAL MATCH path_up = (m)<-[*0..4]-(ancestor)
WITH matched, descendants, collect(DISTINCT ancestor) AS ancestors

// Merge all nodes
WITH matched + descendants + ancestors AS all_raw
UNWIND all_raw AS node
WITH DISTINCT node
WHERE node IS NOT NULL

// Collect relationships between the gathered nodes
WITH collect(node) AS nodes
UNWIND nodes AS a
UNWIND nodes AS b
OPTIONAL MATCH (a)-[r]->(b)
WITH nodes, collect(r) AS rels

RETURN
    [node IN nodes | {
        id: toString(elementId(node)),
        labels: labels(node),
        name: COALESCE(node.name, ""),
        numero: COALESCE(node.numero, ""),
        conteudo: COALESCE(node.conteudo, node.texto, node.ementa, ""),
        topic: COALESCE(node.topic, "")
    }] AS nodes,
    [rel IN rels WHERE rel IS NOT NULL | {
        source: toString(elementId(startNode(rel))),
        target: toString(elementId(endNode(rel))),
        type: type(rel)
    }] AS relationships
"""
