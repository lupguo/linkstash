package entity

import (
	"time"

	"gorm.io/gorm"
)

type ShortLink struct {
	gorm.Model
	Code       string     `gorm:"uniqueIndex;size:16" json:"code"`
	LongURL    string     `gorm:"not null" json:"long_url"`
	ExpiresAt  *time.Time `json:"expires_at"`
	ClickCount int        `gorm:"default:0" json:"click_count"`
}

func (ShortLink) TableName() string { return "t_short_links" }

func (s ShortLink) IsExpired() bool {
	if s.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*s.ExpiresAt)
}
