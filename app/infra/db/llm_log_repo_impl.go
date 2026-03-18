package db

import (
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"gorm.io/gorm"
)

var _ repos.LLMLogRepo = (*LLMLogRepoImpl)(nil)

type LLMLogRepoImpl struct {
	db *gorm.DB
}

func NewLLMLogRepoImpl(db *gorm.DB) *LLMLogRepoImpl {
	return &LLMLogRepoImpl{db: db}
}

func (r *LLMLogRepoImpl) Create(log *entity.LLMLog) error {
	return r.db.Create(log).Error
}

func (r *LLMLogRepoImpl) ListByURLID(urlID uint) ([]*entity.LLMLog, error) {
	var logs []*entity.LLMLog
	if err := r.db.Where("url_id = ?", urlID).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
