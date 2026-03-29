package db

import (
	"time"

	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"gorm.io/gorm"
)

// URLRepoImpl implements repos.URLRepo using GORM.
type URLRepoImpl struct {
	db *gorm.DB
}

// NewURLRepoImpl creates a new URLRepoImpl.
func NewURLRepoImpl(db *gorm.DB) *URLRepoImpl {
	return &URLRepoImpl{db: db}
}

// compile-time interface check
var _ repos.URLRepo = (*URLRepoImpl)(nil)

// Create inserts a new URL record.
func (r *URLRepoImpl) Create(url *entity.URL) error {
	return r.db.Create(url).Error
}

// GetByID retrieves a URL by its primary key.
func (r *URLRepoImpl) GetByID(id uint) (*entity.URL, error) {
	var url entity.URL
	if err := r.db.First(&url, id).Error; err != nil {
		return nil, err
	}
	return &url, nil
}

// GetByLink retrieves a URL by its link value.
func (r *URLRepoImpl) GetByLink(link string) (*entity.URL, error) {
	var url entity.URL
	if err := r.db.Where("link = ?", link).First(&url).Error; err != nil {
		return nil, err
	}
	return &url, nil
}

// GetDeletedByLink retrieves a soft-deleted URL by its link value.
func (r *URLRepoImpl) GetDeletedByLink(link string) (*entity.URL, error) {
	var url entity.URL
	if err := r.db.Unscoped().Where("link = ? AND deleted_at IS NOT NULL", link).First(&url).Error; err != nil {
		return nil, err
	}
	return &url, nil
}

// Restore un-deletes a soft-deleted URL by clearing its deleted_at field.
func (r *URLRepoImpl) Restore(id uint) error {
	return r.db.Unscoped().Model(&entity.URL{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

// Update saves all fields of the given URL record.
func (r *URLRepoImpl) Update(url *entity.URL) error {
	return r.db.Save(url).Error
}

// Delete performs a soft delete on the URL with the given id.
func (r *URLRepoImpl) Delete(id uint) error {
	return r.db.Delete(&entity.URL{}, id).Error
}

// List returns a paginated, sorted, and filtered list of URLs together with the total count.
// sort: "time" orders by created_at DESC; "weight" orders by (auto_weight + manual_weight) DESC.
// category and tags are optional filters (tags uses LIKE matching).
func (r *URLRepoImpl) List(page, size int, sort, category, tags string, isShortURL bool) ([]*entity.URL, int64, error) {
	var urls []*entity.URL
	var total int64

	query := r.db.Model(&entity.URL{})

	// filters
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if tags != "" {
		query = query.Where("tags LIKE ?", "%"+tags+"%")
	}
	if isShortURL {
		query = query.Where("short_code != '' AND short_code IS NOT NULL")
	}

	// total count (before pagination)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// sorting
	switch sort {
	case "weight":
		query = query.Order("auto_weight + manual_weight DESC")
	default: // "time" or empty
		query = query.Order("created_at DESC")
	}

	// pagination
	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	if err := query.Offset(offset).Limit(size).Find(&urls).Error; err != nil {
		return nil, 0, err
	}

	return urls, total, nil
}

// FindByStatus returns all URLs matching the given status.
func (r *URLRepoImpl) FindByStatus(status string) ([]*entity.URL, error) {
	var urls []*entity.URL
	if err := r.db.Where("status = ?", status).Find(&urls).Error; err != nil {
		return nil, err
	}
	return urls, nil
}

// IncrementVisit atomically increments visit_count and auto_weight and updates last_visit_at.
func (r *URLRepoImpl) IncrementVisit(id uint) error {
	return r.db.Model(&entity.URL{}).Where("id = ?", id).Updates(map[string]interface{}{
		"visit_count":   gorm.Expr("visit_count + 1"),
		"auto_weight":   gorm.Expr("auto_weight + 1"),
		"last_visit_at": time.Now(),
	}).Error
}

// GetByShortCode retrieves a URL by its short code.
func (r *URLRepoImpl) GetByShortCode(code string) (*entity.URL, error) {
	var url entity.URL
	if err := r.db.Where("short_code = ?", code).First(&url).Error; err != nil {
		return nil, err
	}
	return &url, nil
}

// ListByShortCode returns a paginated list of URLs that have a short code, ordered by created_at DESC.
func (r *URLRepoImpl) ListByShortCode(page, size int) ([]*entity.URL, int64, error) {
	var urls []*entity.URL
	var total int64

	query := r.db.Model(&entity.URL{}).Where("short_code != '' AND short_code IS NOT NULL")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&urls).Error; err != nil {
		return nil, 0, err
	}

	return urls, total, nil
}

// ClearShortCode removes the short code (and expiration) from a URL without deleting the URL record.
func (r *URLRepoImpl) ClearShortCode(id uint) error {
	return r.db.Model(&entity.URL{}).Where("id = ?", id).Updates(map[string]interface{}{
		"short_code":      "",
		"short_expires_at": nil,
	}).Error
}
