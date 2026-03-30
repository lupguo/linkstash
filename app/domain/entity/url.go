package entity

import (
	"time"

	"gorm.io/gorm"
)

type URL struct {
	ID             uint           `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	Link           string         `gorm:"size:768;uniqueIndex;not null;comment:链接地址" json:"link"`
	Title          string         `gorm:"comment:页面标题" json:"title"`
	Keywords       string         `gorm:"comment:关键词(逗号分隔)" json:"keywords"`
	Description    string         `gorm:"comment:页面描述" json:"description"`
	Category       string         `gorm:"index;comment:分类" json:"category"`
	Tags           string         `gorm:"comment:标签(逗号分隔)" json:"tags"`
	Status         string         `gorm:"default:pending;comment:状态(pending/analyzing/ready/failed)" json:"status"`
	AutoWeight     float64        `gorm:"default:0;comment:自动权重(访问累加)" json:"auto_weight"`
	ManualWeight   float64        `gorm:"default:0;comment:手动权重" json:"manual_weight"`
	LastVisitAt    *time.Time     `gorm:"comment:最近访问时间" json:"last_visit_at"`
	VisitCount     int            `gorm:"default:0;comment:访问次数" json:"visit_count"`
	ShortCode      string         `gorm:"index;size:16;comment:短链接编码" json:"short_code,omitempty"`
	ShortExpiresAt *time.Time     `gorm:"comment:短链接过期时间" json:"short_expires_at,omitempty"`
	Color          string         `gorm:"size:20;default:'';comment:卡片主题色(green/red/cyan/yellow/purple/orange/blue)" json:"color"`
	Icon           string         `gorm:"size:10;default:'';comment:图标(emoji)" json:"icon"`
	Favicon        string         `gorm:"type:mediumtext;comment:网站图标(base64)" json:"favicon,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime;comment:创建时间" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime;comment:更新时间" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index;comment:删除时间(软删除)" json:"deleted_at"`
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
