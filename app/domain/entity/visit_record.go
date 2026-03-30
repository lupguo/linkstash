package entity

import (
	"time"

	"gorm.io/gorm"
)

type VisitRecord struct {
	ID        uint           `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	URLID     uint           `gorm:"index;comment:关联URL的ID" json:"url_id"`
	ShortID   uint           `gorm:"index;comment:短链接ID" json:"short_id"`
	IP        string         `gorm:"comment:访问者IP" json:"ip"`
	UserAgent string         `gorm:"comment:浏览器UA" json:"user_agent"`
	CreatedAt time.Time      `gorm:"autoCreateTime;comment:创建时间" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime;comment:更新时间" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间(软删除)" json:"deleted_at"`
}

func (VisitRecord) TableName() string { return "t_visit_records" }
