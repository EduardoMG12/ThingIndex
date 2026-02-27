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
MATCH (n)-[r]->(m)
WITH n, r, m LIMIT {limit}

// Ensure nodes are unique
WITH collect(distinct n) + collect(distinct m) as all_nodes, collect(r) as all_rels
UNWIND all_nodes as node
WITH distinct node, all_rels

// Group assignment based on Labels for coloring
WITH node, all_rels,
     CASE
         WHEN 'Origem' IN labels(node) THEN 0
         WHEN 'Topic' IN labels(node) THEN 1
         WHEN 'Lei' IN labels(node) THEN 2
         WHEN 'Artigo' IN labels(node) OR 'Paragrafo' IN labels(node) OR 'Inciso' IN labels(node) THEN 3
         ELSE 4
     END as group_id
     
// Identify the node label (e.g., name or id)
WITH node, all_rels, group_id,
     COALESCE(node.name, node.id, split(labels(node)[0], ':')[0]) as node_label
     
WITH collect({{
    id: toString(elementId(node)), 
    label: node_label, 
    group: group_id
}}) as nodes, all_rels

UNWIND all_rels as rel
WITH nodes, collect({{
    source: toString(elementId(startNode(rel))),
    target: toString(elementId(endNode(rel))),
    label: type(rel)
}}) as links

RETURN nodes, links
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
MATCH (n)-[r]->(m)
WHERE (n.topic = $context OR (n:Topic AND n.name = $context) OR (n:Origem AND n.name="Brasil"))
  AND (m.topic = $context OR (m:Topic AND m.name = $context) OR (m:Origem AND m.name="Brasil"))
WITH n, r, m LIMIT {limit}

WITH collect(distinct n) + collect(distinct m) as all_nodes, collect(r) as all_rels
UNWIND all_nodes as node
WITH distinct node, all_rels

WITH node, all_rels,
     CASE
         WHEN 'Origem' IN labels(node) THEN 0
         WHEN 'Topic' IN labels(node) THEN 1
         WHEN 'Lei' IN labels(node) THEN 2
         WHEN 'Artigo' IN labels(node) OR 'Paragrafo' IN labels(node) OR 'Inciso' IN labels(node) THEN 3
         ELSE 4
     END as group_id
     
WITH node, all_rels, group_id,
     COALESCE(node.name, node.id, split(labels(node)[0], ':')[0]) as node_label
     
WITH collect({{
    id: toString(elementId(node)), 
    label: node_label, 
    group: group_id
}}) as nodes, all_rels

UNWIND all_rels as rel
WITH nodes, collect({{
    source: toString(elementId(startNode(rel))),
    target: toString(elementId(endNode(rel))),
    label: type(rel)
}}) as links

RETURN nodes, links
"""
