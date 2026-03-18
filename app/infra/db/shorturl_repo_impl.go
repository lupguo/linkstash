package db

import (
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"gorm.io/gorm"
)

// ShortURLRepoImpl implements repos.ShortURLRepo using GORM.
type ShortURLRepoImpl struct {
	db *gorm.DB
}

// NewShortURLRepoImpl creates a new ShortURLRepoImpl.
func NewShortURLRepoImpl(db *gorm.DB) *ShortURLRepoImpl {
	return &ShortURLRepoImpl{db: db}
}

// compile-time interface check
var _ repos.ShortURLRepo = (*ShortURLRepoImpl)(nil)

// Create inserts a new ShortLink record.
func (r *ShortURLRepoImpl) Create(link *entity.ShortLink) error {
	return r.db.Create(link).Error
}

// GetByCode retrieves a ShortLink by its code.
func (r *ShortURLRepoImpl) GetByCode(code string) (*entity.ShortLink, error) {
	var link entity.ShortLink
	if err := r.db.Where("code = ?", code).First(&link).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

// GetByID retrieves a ShortLink by its primary key.
func (r *ShortURLRepoImpl) GetByID(id uint) (*entity.ShortLink, error) {
	var link entity.ShortLink
	if err := r.db.First(&link, id).Error; err != nil {
		return nil, err
	}
	return &link, nil
}

// Delete performs a soft delete on the ShortLink with the given id.
func (r *ShortURLRepoImpl) Delete(id uint) error {
	return r.db.Delete(&entity.ShortLink{}, id).Error
}

// List returns a paginated list of ShortLinks ordered by created_at DESC.
func (r *ShortURLRepoImpl) List(page, size int) ([]*entity.ShortLink, int64, error) {
	var links []*entity.ShortLink
	var total int64

	query := r.db.Model(&entity.ShortLink{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&links).Error; err != nil {
		return nil, 0, err
	}

	return links, total, nil
}

// IncrementClick atomically increments the click_count for the given ShortLink.
func (r *ShortURLRepoImpl) IncrementClick(id uint) error {
	return r.db.Model(&entity.ShortLink{}).Where("id = ?", id).Update("click_count", gorm.Expr("click_count + 1")).Error
}
