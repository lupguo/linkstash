package entity

import (
	"time"

	"gorm.io/gorm"
)

type URL struct {
	gorm.Model
	Link           string     `gorm:"uniqueIndex;not null" json:"link"`
	Title          string     `json:"title"`
	Keywords       string     `json:"keywords"`
	Description    string     `json:"description"`
	Category       string     `gorm:"index" json:"category"`
	Tags           string     `json:"tags"`
	Status         string     `gorm:"default:pending" json:"status"`
	AutoWeight     float64    `gorm:"default:0" json:"auto_weight"`
	ManualWeight   float64    `gorm:"default:0" json:"manual_weight"`
	LastVisitAt    *time.Time `json:"last_visit_at"`
	VisitCount     int        `gorm:"default:0" json:"visit_count"`
	ShortCode      string     `gorm:"index;size:16" json:"short_code,omitempty"`
	ShortExpiresAt *time.Time `json:"short_expires_at,omitempty"`
	Color          string     `gorm:"size:20;default:''" json:"color"`
	Icon           string     `gorm:"size:10;default:''" json:"icon"`
	Favicon        string     `gorm:"type:text;default:''" json:"favicon,omitempty"`
}

func (URL) TableName() string { return "t_urls" }

// IsShortExpired returns true if the short code has expired.
func (u URL) IsShortExpired() bool {
	if u.ShortExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ShortExpiresAt)
}

// HasShortCode returns true if the URL has a short code assigned.
func (u URL) HasShortCode() bool {
	return u.ShortCode != ""
}
