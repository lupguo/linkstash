package db

import (
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"gorm.io/gorm"
)

var _ repos.EmbeddingRepo = (*EmbeddingRepoImpl)(nil)

type EmbeddingRepoImpl struct {
	db *gorm.DB
}

func NewEmbeddingRepoImpl(db *gorm.DB) *EmbeddingRepoImpl {
	return &EmbeddingRepoImpl{db: db}
}

func (r *EmbeddingRepoImpl) Save(e *entity.Embedding) error {
	return r.db.Where("url_id = ?", e.URLID).Assign(entity.Embedding{Vector: e.Vector}).FirstOrCreate(e).Error
}

func (r *EmbeddingRepoImpl) GetByURLID(urlID uint) (*entity.Embedding, error) {
	var embedding entity.Embedding
	if err := r.db.Where("url_id = ?", urlID).First(&embedding).Error; err != nil {
		return nil, err
	}
	return &embedding, nil
}

func (r *EmbeddingRepoImpl) GetAll() ([]*entity.Embedding, error) {
	var embeddings []*entity.Embedding
	if err := r.db.Find(&embeddings).Error; err != nil {
		return nil, err
	}
	return embeddings, nil
}

func (r *EmbeddingRepoImpl) Delete(urlID uint) error {
	return r.db.Where("url_id = ?", urlID).Delete(&entity.Embedding{}).Error
}
