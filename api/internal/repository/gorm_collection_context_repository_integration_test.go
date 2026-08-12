//go:build integration

package repository

import (
	"context"
	"strconv"
	"testing"

	"thing-index/internal/models"

	"gorm.io/gorm"
)

// Este arquivo cobre GetCollectionContext, que e a traducao em SQL da tese do
// trabalho: contexto documental preservado e reconstituido por modelagem
// relacional, com chave estrangeira, sequence_order e agregacao linear.
//
// Ate aqui essas propriedades eram sustentadas apenas por leitura de codigo.

// A ordem de leitura precisa sair por sequence_order, nao pela ordem de
// insercao. O teste grava os nos embaralhados de proposito: se alguem trocar o
// Order por ordenacao implicita do Postgres, o resultado passa a depender do
// plano de execucao e este teste quebra.
func TestCollectionContextReconstructsStableReadingOrder(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCollectionContextRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "context-order")
	category := seedIntegrationCategory(t, db, user.ID, "Processos")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Vara Civel")
	document := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, "peticao.pdf")

	// Inserido fora de ordem: 3, 1, 4, 2.
	for _, order := range []int{3, 1, 4, 2} {
		mustCreateRecord(t, db, &models.ParagraphNode{
			DocumentID:    document.ID,
			SequenceOrder: order,
			Content:       paragraphContentFor(order),
		})
	}

	result, err := repo.GetCollectionContext(ctx, user.ID, collection.ID)
	if err != nil {
		t.Fatalf("GetCollectionContext: %v", err)
	}

	if len(result.ParagraphNodes) != 4 {
		t.Fatalf("paragrafos = %d, quero 4", len(result.ParagraphNodes))
	}

	for index, node := range result.ParagraphNodes {
		wantOrder := index + 1
		if node.SequenceOrder != wantOrder {
			t.Fatalf("posicao %d tem sequence_order %d, quero %d", index, node.SequenceOrder, wantOrder)
		}
		if node.Content != paragraphContentFor(wantOrder) {
			t.Fatalf("posicao %d tem conteudo %q, quero %q", index, node.Content, paragraphContentFor(wantOrder))
		}
	}
}

// Com varios documentos na mesma collection, a ordenacao e composta:
// primeiro document_id, depois sequence_order. Sem a primeira chave, os
// paragrafos de documentos diferentes se intercalam e o contexto entregue ao
// LLM vira uma mistura de duas pecas processuais.
func TestCollectionContextGroupsNodesByDocumentBeforeSequence(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCollectionContextRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "context-grouping")
	category := seedIntegrationCategory(t, db, user.ID, "Processos")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Comarca")

	first := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, "peticao-inicial.pdf")
	second := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, "contestacao.pdf")

	// Grava o segundo documento antes do primeiro, e dentro de cada um grava a
	// ordem invertida.
	for _, documentID := range []uint{second.ID, first.ID} {
		for _, order := range []int{2, 1} {
			mustCreateRecord(t, db, &models.ParagraphNode{
				DocumentID:    documentID,
				SequenceOrder: order,
				Content:       paragraphContentFor(order),
			})
		}
	}

	result, err := repo.GetCollectionContext(ctx, user.ID, collection.ID)
	if err != nil {
		t.Fatalf("GetCollectionContext: %v", err)
	}

	if len(result.ParagraphNodes) != 4 {
		t.Fatalf("paragrafos = %d, quero 4", len(result.ParagraphNodes))
	}

	wantDocuments := []uint{first.ID, first.ID, second.ID, second.ID}
	wantOrders := []int{1, 2, 1, 2}
	for index, node := range result.ParagraphNodes {
		if node.DocumentID != wantDocuments[index] {
			t.Fatalf("posicao %d veio do documento %d, quero %d", index, node.DocumentID, wantDocuments[index])
		}
		if node.SequenceOrder != wantOrders[index] {
			t.Fatalf("posicao %d tem sequence_order %d, quero %d", index, node.SequenceOrder, wantOrders[index])
		}
	}

	// Os documentos tambem precisam sair em ordem estavel.
	if len(result.Documents) != 2 || result.Documents[0].ID != first.ID {
		t.Fatalf("documentos fora de ordem: %+v", result.Documents)
	}
}

// Isolamento entre usuarios. As consultas das tabelas derivadas filtram apenas
// por document_id, sem user_id — a protecao vem de os IDs terem sido obtidos
// numa consulta ja escopada por usuario. E uma garantia transitiva, e portanto
// fragil a refatoracao: basta alguem trocar a origem da lista de IDs para o
// vazamento aparecer sem nenhum erro de compilacao.
func TestCollectionContextRejectsCollectionOfAnotherUser(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCollectionContextRepository{db: db}
	ctx := context.Background()

	owner := seedIntegrationUser(t, db, "context-owner")
	intruder := seedIntegrationUser(t, db, "context-intruder")

	category := seedIntegrationCategory(t, db, owner.ID, "Confidencial")
	collection := seedIntegrationCollection(t, db, owner.ID, category.ID, "Sigiloso")
	document := seedIntegrationDocument(t, db, owner.ID, category.ID, collection.ID, "sigiloso.pdf")
	seedIntegrationDerivedData(t, db, document.ID)

	result, err := repo.GetCollectionContext(ctx, intruder.ID, collection.ID)
	if err == nil {
		t.Fatalf("usuario alheio recuperou a collection: %+v", result)
	}
	if result != nil {
		t.Fatalf("esperava contexto nulo no erro, veio %+v", result)
	}
}

// Um documento de outro usuario nunca pode entrar no contexto, mesmo quando o
// solicitante e dono da collection.
func TestCollectionContextExcludesDocumentsOfAnotherUser(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCollectionContextRepository{db: db}
	ctx := context.Background()

	owner := seedIntegrationUser(t, db, "context-mixed-owner")
	intruder := seedIntegrationUser(t, db, "context-mixed-intruder")

	category := seedIntegrationCategory(t, db, owner.ID, "Compartilhada")
	collection := seedIntegrationCollection(t, db, owner.ID, category.ID, "Mista")

	ownDocument := seedIntegrationDocument(t, db, owner.ID, category.ID, collection.ID, "meu.pdf")
	seedIntegrationDerivedData(t, db, ownDocument.ID)

	// Documento de outro usuario apontando para a mesma collection.
	foreignCategory := seedIntegrationCategory(t, db, intruder.ID, "Alheia")
	foreignDocument := seedIntegrationDocument(t, db, intruder.ID, foreignCategory.ID, collection.ID, "alheio.pdf")
	seedIntegrationDerivedData(t, db, foreignDocument.ID)

	result, err := repo.GetCollectionContext(ctx, owner.ID, collection.ID)
	if err != nil {
		t.Fatalf("GetCollectionContext: %v", err)
	}

	if len(result.Documents) != 1 || result.Documents[0].ID != ownDocument.ID {
		t.Fatalf("contexto trouxe documento alheio: %+v", result.Documents)
	}

	for _, node := range result.ParagraphNodes {
		if node.DocumentID == foreignDocument.ID {
			t.Fatal("paragrafo de documento alheio vazou para o contexto")
		}
	}
	for _, chunk := range result.Chunks {
		if chunk.DocumentID == foreignDocument.ID {
			t.Fatal("chunk de documento alheio vazou para o contexto")
		}
	}
	for _, pair := range result.MetadataPairs {
		if pair.DocumentID == foreignDocument.ID {
			t.Fatal("metadata de documento alheio vazou para o contexto")
		}
	}
}

// A reconstrucao precisa devolver tudo que foi indexado, sem perder nem
// duplicar no. Duplicacao e o risco real aqui: um JOIN mal escrito no lugar das
// consultas por tabela multiplicaria linhas em produto cartesiano.
func TestCollectionContextReconstructionIsFaithful(t *testing.T) {
	db := openIntegrationDB(t)
	repo := &GormCollectionContextRepository{db: db}
	ctx := context.Background()

	user := seedIntegrationUser(t, db, "context-fidelity")
	category := seedIntegrationCategory(t, db, user.ID, "Processos")
	collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Fidelidade")

	const documentCount = 3
	for index := 0; index < documentCount; index++ {
		document := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, documentTitleFor(index))
		seedIntegrationDerivedData(t, db, document.ID)
	}

	result, err := repo.GetCollectionContext(ctx, user.ID, collection.ID)
	if err != nil {
		t.Fatalf("GetCollectionContext: %v", err)
	}

	// seedIntegrationDerivedData grava exatamente um registro de cada tipo.
	checks := []struct {
		name string
		got  int
	}{
		{"documentos", len(result.Documents)},
		{"paginas", len(result.DocumentPages)},
		{"paragrafos", len(result.ParagraphNodes)},
		{"tabelas", len(result.TableSections)},
		{"chunks", len(result.Chunks)},
		{"metadata", len(result.MetadataPairs)},
		{"outline", len(result.OutlineNodes)},
	}
	for _, check := range checks {
		if check.got != documentCount {
			t.Fatalf("%s = %d, quero %d (perda ou duplicacao na reconstrucao)", check.name, check.got, documentCount)
		}
	}

	if result.Collection == nil || result.Collection.ID != collection.ID {
		t.Fatalf("collection ausente no contexto: %+v", result.Collection)
	}
	if result.Collection.Category == nil {
		t.Fatal("categoria nao veio no preload")
	}
}

// A afirmacao de agregacao linear do artigo: o custo de consulta nao cresce com
// o numero de documentos. GetCollectionContext faz um numero FIXO de consultas,
// entao 1 documento e 20 documentos custam a mesma quantidade de round trips.
//
// Sao 9 no total:
//
//	1 collection + 1 preload da categoria + 1 documentos + 6 tabelas derivadas
//	  (pages, paragraph_nodes, table_sections, chunks, metadata_pairs, outline)
//
// Este e o teste que separa a arquitetura relacional de uma implementacao
// ingenua com N+1: sem ele, "agregacao linear" e so uma frase no texto.
//
// O limite superior e proposital. Se alguem adicionar uma consulta por
// documento, a primeira assercao pega; se adicionar uma consulta fixa nova, a
// segunda obriga a revisar este comentario e o numero citado no artigo.
const collectionContextQueryBudget = 9

func TestCollectionContextQueryCountDoesNotGrowWithDocuments(t *testing.T) {
	db := openIntegrationDB(t)
	ctx := context.Background()

	countQueriesFor := func(t *testing.T, suffix string, documentCount int) int {
		t.Helper()

		user := seedIntegrationUser(t, db, "context-queries-"+suffix)
		category := seedIntegrationCategory(t, db, user.ID, "Processos "+suffix)
		collection := seedIntegrationCollection(t, db, user.ID, category.ID, "Escala "+suffix)

		for index := 0; index < documentCount; index++ {
			document := seedIntegrationDocument(t, db, user.ID, category.ID, collection.ID, documentTitleFor(index))
			seedIntegrationDerivedData(t, db, document.ID)
		}

		counted, queries := newQueryCountingDB(t, db)
		repo := &GormCollectionContextRepository{db: counted}

		if _, err := repo.GetCollectionContext(ctx, user.ID, collection.ID); err != nil {
			t.Fatalf("GetCollectionContext: %v", err)
		}

		return *queries
	}

	small := countQueriesFor(t, "small", 1)
	large := countQueriesFor(t, "large", 20)

	if small != large {
		t.Fatalf("consultas cresceram com o numero de documentos: %d para 1 documento, %d para 20 (sintoma classico de N+1)", small, large)
	}
	if large != collectionContextQueryBudget {
		t.Fatalf("consultas fixas = %d, quero %d; se a mudanca for intencional, atualize a constante e o numero citado no artigo", large, collectionContextQueryBudget)
	}

	t.Logf("agregacao linear confirmada: %d consultas para 1 documento e %d para 20", small, large)
}

// newQueryCountingDB devolve uma sessao do gorm que conta cada SQL emitido.
func newQueryCountingDB(t *testing.T, db *gorm.DB) (*gorm.DB, *int) {
	t.Helper()

	queries := 0
	session := db.Session(&gorm.Session{NewDB: true})
	count := func(tx *gorm.DB) { queries++ }

	if err := session.Callback().Query().After("gorm:query").Register("integration:count_query", count); err != nil {
		t.Fatalf("registrar contagem de query: %v", err)
	}
	if err := session.Callback().Row().After("gorm:row").Register("integration:count_row", count); err != nil {
		t.Fatalf("registrar contagem de row: %v", err)
	}
	if err := session.Callback().Raw().After("gorm:raw").Register("integration:count_raw", count); err != nil {
		t.Fatalf("registrar contagem de raw: %v", err)
	}

	return session, &queries
}

func paragraphContentFor(order int) string {
	return "paragrafo na posicao " + strconv.Itoa(order)
}

func documentTitleFor(index int) string {
	return "documento-" + strconv.Itoa(index) + ".pdf"
}
