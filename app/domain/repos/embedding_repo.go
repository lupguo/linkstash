package repos

import "github.com/lupguo/linkstash/app/domain/entity"

type EmbeddingRepo interface {
	Save(embedding *entity.Embedding) error
	GetByURLID(urlID uint) (*entity.Embedding, error)
	GetAll() ([]*entity.Embedding, error)
	Delete(urlID uint) error
}
