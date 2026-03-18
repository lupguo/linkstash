package repos

import "github.com/lupguo/linkstash/app/domain/entity"

type ShortURLRepo interface {
	Create(shortLink *entity.ShortLink) error
	GetByCode(code string) (*entity.ShortLink, error)
	GetByID(id uint) (*entity.ShortLink, error)
	Delete(id uint) error
	List(page int, size int) ([]*entity.ShortLink, int64, error)
	IncrementClick(id uint) error
}
