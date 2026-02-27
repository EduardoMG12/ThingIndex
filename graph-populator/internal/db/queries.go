package db

// QueryEnsureConstraints contains a list of queries to create unique constraints and indexes
var QueryEnsureConstraints = []string{
	"CREATE CONSTRAINT IF NOT EXISTS FOR (l:Lei) REQUIRE l.id IS UNIQUE",
	"CREATE INDEX IF NOT EXISTS FOR (l:Lei) ON (l.numero)",
	"CREATE CONSTRAINT IF NOT EXISTS FOR (a:Artigo) REQUIRE a.id IS UNIQUE",
	"CREATE CONSTRAINT IF NOT EXISTS FOR (p:Paragrafo) REQUIRE p.id IS UNIQUE",
	"CREATE CONSTRAINT IF NOT EXISTS FOR (i:Inciso) REQUIRE i.id IS UNIQUE",
	"CREATE CONSTRAINT IF NOT EXISTS FOR (al:Alinea) REQUIRE al.id IS UNIQUE",
}

// QueryCreateLaw creates a Law node with all its properties
const QueryCreateLaw = `
	MERGE (l:Lei {id: $id})
	SET l.tipo = $tipo,
		l.esfera = $esfera,
		l.numero = $numero,
		l.ano = $ano,
		l.titulo = $titulo,
		l.ementa = $ementa,
		l.url_origem = $url_origem,
		l.url_pai = $url_pai
`

// QueryCreateArticle creates an Article node and links it to its parent Law
const QueryCreateArticle = `
	MATCH (l:Lei {id: $lawID})
	MERGE (a:Artigo {id: $id})
	SET a.numero = $numero,
		a.caput = $caput,
		a.status = $status
	MERGE (l)-[:CONTEM]->(a)
`

// QueryCreateParagraph creates a Paragraph node and links it to its parent Article
const QueryCreateParagraph = `
	MATCH (a:Artigo {id: $artID})
	MERGE (p:Paragrafo {id: $id})
	SET p.numero = $numero,
		p.texto = $texto,
		p.status = $status
	MERGE (a)-[:CONTEM]->(p)
`

// QueryCreateInciso creates an Inciso node and links it to a dynamic parent (Article or Paragraph)
const QueryCreateInciso = `
	MATCH (p:%s {id: $parentID})
	MERGE (i:Inciso {id: $id})
	SET i.numero = $numero,
		i.texto = $texto,
		i.status = $status
	MERGE (p)-[:CONTEM]->(i)
`

// QueryCreateAlinea creates an Alinea node and links it to its parent Inciso
const QueryCreateAlinea = `
	MATCH (i:Inciso {id: $incID})
	MERGE (al:Alinea {id: $id})
	SET al.numero = $numero,
		al.texto = $texto,
		al.status = $status
	MERGE (i)-[:CONTEM]->(al)
`

// QueryLinkArticleToLaw connects an Article to another Law it cites
const QueryLinkArticleToLaw = `
	MATCH (a:Artigo {id: $articleID})
	MERGE (target:Lei {numero: $targetNum})
	ON CREATE SET target.tipo = $targetType, target.esfera = 'federal', target.status = 'unknown'
	MERGE (a)-[:CITA]->(target)
`

// --- Verifier Queries ---

// QueryCountNodesByLabel returns the count of nodes grouped by their primary label
const QueryCountNodesByLabel = "MATCH (n) RETURN head(labels(n)) as Label, count(n) as Count ORDER BY Count DESC"

// QueryCountRelationshipsByType returns the count of relationships grouped by their type
const QueryCountRelationshipsByType = "MATCH ()-[r]->() RETURN type(r) as Type, count(r) as Count ORDER BY Count DESC"

// QuerySampleHierarchy returns the top 5 laws with the most articles
const QuerySampleHierarchy = `
	MATCH (l:Lei)-[:CONTEM]->(a:Artigo)
	RETURN l.numero as Lei, count(a) as Artigos
	ORDER BY Artigos DESC LIMIT 5
`

// QuerySampleDeepestPath returns 5 examples of the deepest hierarchical path in the graph
const QuerySampleDeepestPath = `
	MATCH (l:Lei)-[:CONTEM]->(a:Artigo)-[:CONTEM]->(p:Paragrafo)-[:CONTEM]->(i:Inciso)-[:CONTEM]->(al:Alinea)
	RETURN l.numero as Lei, a.numero as Art, p.numero as Par, i.numero as Inc, al.numero as Ali
	LIMIT 5
`

// QuerySampleCrossReferences returns 10 examples of laws citing other laws
const QuerySampleCrossReferences = `
	MATCH (l1:Lei)-[:CONTEM]->(a:Artigo)-[:CITA]->(l2:Lei)
	RETURN l1.numero as Lei_Origem, a.numero as Artigo, l2.numero as Lei_Citada
	LIMIT 10
`

// QueryGhostNodes finds laws that are referenced but not yet ingested
const QueryGhostNodes = `
	MATCH (l:Lei)
	WHERE l.titulo IS NULL AND l.status = 'unknown'
	RETURN l.numero as Lei_Fantasma, count(l)
	LIMIT 10
`
