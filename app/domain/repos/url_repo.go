package repos

import "github.com/lupguo/linkstash/app/domain/entity"

type URLRepo interface {
	Create(url *entity.URL) error
	GetByID(id uint) (*entity.URL, error)
	GetByLink(link string) (*entity.URL, error)
	GetDeletedByLink(link string) (*entity.URL, error)
	Restore(id uint) error
	Update(url *entity.URL) error
	Delete(id uint) error
	List(page int, size int, sort string, category string, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error)
	FindByStatus(status string) ([]*entity.URL, error)
	IncrementVisit(id uint) error

	// Short code related methods
	GetByShortCode(code string) (*entity.URL, error)
	ListByShortCode(page, size int) ([]*entity.URL, int64, error)
	ClearShortCode(id uint) error
}
