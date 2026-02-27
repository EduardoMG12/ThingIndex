package graph

// Centralized Cypher queries for the Populator Neo4j Client

// QueryCreateTopic creates the Origin and Topic nodes and links them
const QueryCreateTopic = `
	MERGE (o:Origem {name: "Brasil"})
	MERGE (t:Topic {name: $topic})
	MERGE (t)-[:PERTENCE]->(o)
`

// QueryCreateDocLinkedToParent creates a document and links it to its parent document
const QueryCreateDocLinkedToParent = `
	MERGE (d:LegalDocument:%s:%s {id: $id})
	MERGE (p:LegalDocument {id: $parent_id})
	MERGE (d)-[:CONTIDO_EM]->(p)
	SET d.tipo      = $tipo,
		d.esfera    = $esfera,
		d.numero    = $numero,
		d.ano       = $ano,
		d.titulo    = $titulo,
		d.ementa    = $ementa,
		d.url_origem = $url_origem,
		d.url_pai   = $url_pai,
		d.artigos   = $artigos,
		d.topic     = $topic
`

// QueryCreateDocLinkedToTopic creates a document at the root level (no parent) and links it directly to the Topic
const QueryCreateDocLinkedToTopic = `
	MERGE (t:Topic {name: $topic})
	MERGE (d:LegalDocument:%s:%s {id: $id})
	MERGE (d)-[:PERTENCE]->(t)
	SET d.tipo      = $tipo,
		d.esfera    = $esfera,
		d.numero    = $numero,
		d.ano       = $ano,
		d.titulo    = $titulo,
		d.ementa    = $ementa,
		d.url_origem = $url_origem,
		d.url_pai   = $url_pai,
		d.artigos   = $artigos,
		d.topic     = $topic
`

// QueryCreateReference links a source document to a target document
const QueryCreateReference = `
	MATCH (source:LegalDocument {id: $source_id})
	MERGE (target:LegalDocument:Documento {id: $target_id})
	MERGE (source)-[r:REFERENCIA]->(target)
	SET r.artigo    = $artigo,
		r.paragrafo = $paragrafo,
		r.inciso    = $inciso,
		r.alinea    = $alinea
`

// QueryCreateArticle creates an Article node and links it to its parent Law
const QueryCreateArticle = `
	MATCH (l:LegalDocument {id: $lawID})
	MERGE (a:Artigo {id: $id})
	SET a.numero = $numero,
		a.caput = $caput,
		a.status = $status,
		a.topic = $topic
	MERGE (l)-[:CONTEM]->(a)
`

// QueryCreateParagraph creates a Paragraph node and links it to its parent Article
const QueryCreateParagraph = `
	MATCH (a:Artigo {id: $artID})
	MERGE (p:Paragrafo {id: $id})
	SET p.numero = $numero,
		p.texto = $texto,
		p.status = $status,
		p.topic = $topic
	MERGE (a)-[:CONTEM]->(p)
`

// QueryCreateInciso creates an Inciso node and links it to a dynamic parent (Article or Paragraph)
const QueryCreateInciso = `
	MATCH (p:%s {id: $parentID})
	MERGE (i:Inciso {id: $id})
	SET i.numero = $numero,
		i.texto = $texto,
		i.status = $status,
		i.topic = $topic
	MERGE (p)-[:CONTEM]->(i)
`

// QueryCreateAlinea creates an Alinea node and links it to its parent Inciso
const QueryCreateAlinea = `
	MATCH (i:Inciso {id: $incID})
	MERGE (al:Alinea {id: $id})
	SET al.numero = $numero,
		al.texto = $texto,
		al.status = $status,
		al.topic = $topic
	MERGE (i)-[:CONTEM]->(al)
`
