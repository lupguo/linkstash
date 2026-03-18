package repos

import "github.com/lupguo/linkstash/app/domain/entity"

type LLMLogRepo interface {
	Create(log *entity.LLMLog) error
	ListByURLID(urlID uint) ([]*entity.LLMLog, error)
}
