package repository

import (
	"context"
	"log"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

// Tetos de materializacao do contexto de colecao.
//
// O problema que eles resolvem: GetCollectionContext e chamado a CADA mensagem
// de chat e carregava para a RAM o texto integral de todas as paginas,
// paragrafos, tabelas, chunks e metadados da colecao inteira -- para depois o
// prompt usar no maximo 6 fontes. Uma colecao grande derrubava o processo por
// OOM, e o custo era pago em toda pergunta.
//
// Os limites nao alteram o numero de consultas: continuam sendo as mesmas 9
// consultas de custo constante que o artigo reporta e que
// TestCollectionContextQueryCountDoesNotGrowWithDocuments trava. O que muda e o
// VOLUME materializado por consulta, que passa a ter teto.
//
// Os valores sao folgados de proposito. O objetivo e evitar que uma colecao
// patologica derrube o processo, nao apertar o caso comum: com ~60 documentos
// de 20 paginas, nenhum teto e alcancado. Quando um teto e atingido, o fato e
// registrado em log -- truncar em silencio degradaria a qualidade da resposta
// sem deixar rastro, que e exatamente o tipo de defeito invisivel que este
// projeto vem eliminando.
const (
	maxCollectionContextPages          = 5000
	maxCollectionContextParagraphNodes = 20000
	maxCollectionContextTableSections  = 5000
	maxCollectionContextChunks         = 20000
	maxCollectionContextMetadataPairs  = 10000
	maxCollectionContextOutlineNodes   = 10000
)

type GormCollectionContextRepository struct {
	db *gorm.DB
}

func NewGormCollectionContextRepository(db *gorm.DB) domain.CollectionContextRepository {
	return &GormCollectionContextRepository{db: db}
}

func (r *GormCollectionContextRepository) GetCollectionContext(ctx context.Context, userID uint, collectionID uint) (*domain.CollectionContext, error) {
	var collection models.Collection
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Where("id = ? AND user_id = ?", collectionID, userID).
		First(&collection).Error; err != nil {
		return nil, err
	}

	var documents []models.Document
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND collection_id = ?", userID, collectionID).
		Order("id ASC").
		Find(&documents).Error; err != nil {
		return nil, err
	}

	documentIDs := make([]uint, 0, len(documents))
	for _, doc := range documents {
		documentIDs = append(documentIDs, doc.ID)
	}

	documentPages := make([]models.DocumentPage, 0)
	paragraphNodes := make([]models.ParagraphNode, 0)
	tableSections := make([]models.TableSection, 0)
	chunks := make([]models.DocumentChunk, 0)
	metadataPairs := make([]models.MetadataPair, 0)
	outlineNodes := make([]models.DocumentOutlineNode, 0)

	if len(documentIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, page_number ASC").
			Limit(maxCollectionContextPages).
			Find(&documentPages).Error; err != nil {
			return nil, err
		}
		warnIfTruncated(collectionID, "document_pages", len(documentPages), maxCollectionContextPages)

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Limit(maxCollectionContextParagraphNodes).
			Find(&paragraphNodes).Error; err != nil {
			return nil, err
		}
		warnIfTruncated(collectionID, "paragraph_nodes", len(paragraphNodes), maxCollectionContextParagraphNodes)

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Limit(maxCollectionContextTableSections).
			Find(&tableSections).Error; err != nil {
			return nil, err
		}
		warnIfTruncated(collectionID, "table_sections", len(tableSections), maxCollectionContextTableSections)

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Limit(maxCollectionContextChunks).
			Find(&chunks).Error; err != nil {
			return nil, err
		}
		warnIfTruncated(collectionID, "document_chunks", len(chunks), maxCollectionContextChunks)

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Limit(maxCollectionContextMetadataPairs).
			Find(&metadataPairs).Error; err != nil {
			return nil, err
		}
		warnIfTruncated(collectionID, "metadata_pairs", len(metadataPairs), maxCollectionContextMetadataPairs)

		if err := r.db.WithContext(ctx).
			Where("document_id IN ?", documentIDs).
			Order("document_id ASC, sequence_order ASC").
			Limit(maxCollectionContextOutlineNodes).
			Find(&outlineNodes).Error; err != nil {
			return nil, err
		}
		warnIfTruncated(collectionID, "document_outline_nodes", len(outlineNodes), maxCollectionContextOutlineNodes)
	}

	return &domain.CollectionContext{
		Collection:     &collection,
		Documents:      documents,
		DocumentPages:  documentPages,
		ParagraphNodes: paragraphNodes,
		TableSections:  tableSections,
		Chunks:         chunks,
		MetadataPairs:  metadataPairs,
		OutlineNodes:   outlineNodes,
	}, nil
}

// warnIfTruncated registra quando um teto foi atingido.
//
// Alcancar o limite significa que parte da colecao ficou de fora da
// reconstrucao, e portanto invisivel para a recuperacao daquela consulta. Isso
// precisa aparecer em log: para um sistema cuja tese e auditabilidade, truncar
// evidencia em silencio e o pior resultado possivel.
func warnIfTruncated(collectionID uint, table string, loaded int, limit int) {
	if loaded < limit {
		return
	}

	log.Printf(
		"collection context: colecao %d atingiu o teto de %d registros em %s; a reconstrucao esta truncada e a resposta pode ignorar evidencia",
		collectionID, limit, table,
	)
}
