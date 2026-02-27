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
       WHEN 'Lei' IN labels(n) OR 'LEI' IN labels(n) OR 'Lei' IN labels(m) OR 'LEI' IN labels(m) THEN 2
       ELSE 3
     END as priority
ORDER BY priority ASC
LIMIT {limit}

// Ensure nodes are unique
WITH collect(distinct n) + collect(distinct m) as all_nodes, collect(r) as all_rels

RETURN 
    [node IN all_nodes | {{
        id: toString(elementId(node)), 
        label: COALESCE(node.name, node.id, split(labels(node)[0], ':')[0]), 
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
       WHEN 'Lei' IN labels(n) OR 'LEI' IN labels(n) OR 'Lei' IN labels(m) OR 'LEI' IN labels(m) THEN 2
       ELSE 3
     END as priority
ORDER BY priority ASC
LIMIT {limit}

WITH collect(distinct n) + collect(distinct m) as all_nodes, collect(r) as all_rels

RETURN 
    [node IN all_nodes | {{
        id: toString(elementId(node)), 
        label: COALESCE(node.name, node.id, split(labels(node)[0], ':')[0]), 
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
