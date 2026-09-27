package usecase

import (
	"context"
	"errors"
	"testing"

	"thing-index/internal/domain"
	"thing-index/internal/models"

	"gorm.io/gorm"
)

type chatContextoProibido struct{}

func (chatContextoProibido) GetCollectionContext(context.Context, uint, uint) (*domain.CollectionContext, error) {
	return nil, errors.New("CreateConversation nao deveria materializar a colecao")
}

type chatColecaoPorPosse struct {
	colecao        *models.Collection
	err            error
	usuarioChecado uint
}

func (c *chatColecaoPorPosse) GetCollectionByID(_ context.Context, _ uint, userID uint) (*models.Collection, error) {
	c.usuarioChecado = userID
	return c.colecao, c.err
}

// Regressao P2-1: CreateConversation chamava GetCollectionContext (9 consultas,
// texto integral de toda a colecao) so para checar posse e jogava fora.
func TestCreateConversationChecaPosseSemMaterializarAColecao(t *testing.T) {
	t.Parallel()

	colecoes := &chatColecaoPorPosse{colecao: &models.Collection{ID: 3, UserID: 7}}
	repo := &chatConversationRepoStub{}
	useCase := NewChatUseCase(colecoes, chatContextoProibido{}, repo, &chatLLMResolverStub{}, nil, DefaultRetrievalOptions())

	conversa, err := useCase.CreateConversation(context.Background(), 7, 3, "titulo")
	if err != nil {
		t.Fatalf("CreateConversation falhou: %v", err)
	}
	if conversa.CollectionID != 3 || conversa.UserID != 7 {
		t.Fatalf("conversa inesperada: %+v", conversa)
	}
	if colecoes.usuarioChecado != 7 {
		t.Fatalf("posse checada para o usuario errado: %d", colecoes.usuarioChecado)
	}
}

func TestCreateConversationDevolveColecaoNaoEncontradaParaColecaoDeOutroUsuario(t *testing.T) {
	t.Parallel()

	colecoes := &chatColecaoPorPosse{err: gorm.ErrRecordNotFound}
	useCase := NewChatUseCase(colecoes, chatContextoProibido{}, &chatConversationRepoStub{}, &chatLLMResolverStub{}, nil, DefaultRetrievalOptions())

	if _, err := useCase.CreateConversation(context.Background(), 8, 3, "titulo"); !errors.Is(err, domain.ErrCollectionNotFound) {
		t.Fatalf("esperava ErrCollectionNotFound, veio %v", err)
	}
}
