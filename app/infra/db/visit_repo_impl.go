package db

import (
	"github.com/lupguo/linkstash/app/domain/entity"
	"github.com/lupguo/linkstash/app/domain/repos"
	"gorm.io/gorm"
)

// VisitRepoImpl implements repos.VisitRepo using GORM.
type VisitRepoImpl struct {
	db *gorm.DB
}

// NewVisitRepoImpl creates a new VisitRepoImpl.
func NewVisitRepoImpl(db *gorm.DB) *VisitRepoImpl {
	return &VisitRepoImpl{db: db}
}

// compile-time interface check
var _ repos.VisitRepo = (*VisitRepoImpl)(nil)

// Create inserts a new VisitRecord.
func (r *VisitRepoImpl) Create(record *entity.VisitRecord) error {
	return r.db.Create(record).Error
}

// ListByURLID returns a paginated list of visit records for the given URL ID.
func (r *VisitRepoImpl) ListByURLID(urlID uint, page, size int) ([]*entity.VisitRecord, int64, error) {
	var records []*entity.VisitRecord
	var total int64

	query := r.db.Model(&entity.VisitRecord{}).Where("url_id = ?", urlID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListByShortID returns a paginated list of visit records for the given short link ID.
func (r *VisitRepoImpl) ListByShortID(shortID uint, page, size int) ([]*entity.VisitRecord, int64, error) {
	var records []*entity.VisitRecord
	var total int64

	query := r.db.Model(&entity.VisitRecord{}).Where("short_id = ?", shortID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if offset < 0 {
		offset = 0
	}
	if err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}
