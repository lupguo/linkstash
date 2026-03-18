package entity

import "gorm.io/gorm"

type VisitRecord struct {
	gorm.Model
	URLID     uint   `gorm:"index" json:"url_id"`
	ShortID   uint   `gorm:"index" json:"short_id"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

func (VisitRecord) TableName() string { return "t_visit_records" }
