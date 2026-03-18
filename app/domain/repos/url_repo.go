package repos

import "github.com/lupguo/linkstash/app/domain/entity"

type URLRepo interface {
	Create(url *entity.URL) error
	GetByID(id uint) (*entity.URL, error)
	GetByLink(link string) (*entity.URL, error)
	Update(url *entity.URL) error
	Delete(id uint) error
	List(page int, size int, sort string, category string, tags string) ([]*entity.URL, int64, error)
	FindByStatus(status string) ([]*entity.URL, error)
	IncrementVisit(id uint) error
}
