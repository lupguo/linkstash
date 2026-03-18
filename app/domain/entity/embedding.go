package entity

import "gorm.io/gorm"

type Embedding struct {
	gorm.Model
	URLID  uint   `gorm:"uniqueIndex;not null" json:"url_id"`
	Vector []byte `json:"-"` // 512-dim float32 vector serialized as []byte (BLOB), ~2KB per record
}

func (Embedding) TableName() string { return "t_embeddings" }
