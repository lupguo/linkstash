package entity

import (
	"time"

	"gorm.io/gorm"
)

type Embedding struct {
	ID        uint           `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	URLID     uint           `gorm:"uniqueIndex;not null;comment:关联URL的ID" json:"url_id"`
	Vector    []byte         `gorm:"comment:向量数据(float32 BLOB)" json:"-"` // 512-dim float32 vector serialized as []byte (BLOB), ~2KB per record
	CreatedAt time.Time      `gorm:"autoCreateTime;comment:创建时间" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime;comment:更新时间" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间(软删除)" json:"deleted_at"`
}

func (Embedding) TableName() string { return "t_embeddings" }
