package entity

import (
	"time"

	"gorm.io/gorm"
)

type URL struct {
	gorm.Model
	Link         string     `gorm:"uniqueIndex;not null" json:"link"`
	Title        string     `json:"title"`
	Keywords     string     `json:"keywords"`
	Description  string     `json:"description"`
	Category     string     `gorm:"index" json:"category"`
	Tags         string     `json:"tags"`
	Status       string     `gorm:"default:pending" json:"status"`
	AutoWeight   float64    `gorm:"default:0" json:"auto_weight"`
	ManualWeight float64    `gorm:"default:0" json:"manual_weight"`
	LastVisitAt  *time.Time `json:"last_visit_at"`
	VisitCount   int        `gorm:"default:0" json:"visit_count"`
}

func (URL) TableName() string { return "t_urls" }
