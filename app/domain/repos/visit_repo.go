package repos

import "github.com/lupguo/linkstash/app/domain/entity"

type VisitRepo interface {
	Create(record *entity.VisitRecord) error
	ListByURLID(urlID uint, page int, size int) ([]*entity.VisitRecord, int64, error)
	ListByShortID(shortID uint, page int, size int) ([]*entity.VisitRecord, int64, error)
}
