package entity

import (
	"time"

	"gorm.io/gorm"
)

type LLMLog struct {
	ID            uint           `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	URLID         uint           `gorm:"index;comment:关联URL的ID" json:"url_id"`
	RequestType   string         `gorm:"index;comment:请求类型(chat/embedding)" json:"request_type"`
	Provider      string         `gorm:"comment:LLM服务商" json:"provider"`
	ModelName     string         `gorm:"column:model;comment:模型名称" json:"model"`
	PromptKey     string         `gorm:"comment:提示词标识" json:"prompt_key"`
	InputContent  string         `gorm:"type:mediumtext;comment:输入内容" json:"input_content"`
	OutputContent string         `gorm:"type:mediumtext;comment:输出内容" json:"output_content"`
	InputTokens   int            `gorm:"comment:输入Token数" json:"input_tokens"`
	OutputTokens  int            `gorm:"comment:输出Token数" json:"output_tokens"`
	TotalTokens   int            `gorm:"comment:总Token数" json:"total_tokens"`
	LatencyMs     int64          `gorm:"comment:响应延迟(毫秒)" json:"latency_ms"`
	TokensPerSec  float64        `gorm:"comment:吞吐量(Token/秒)" json:"tokens_per_sec"`
	StatusCode    int            `gorm:"comment:HTTP状态码" json:"status_code"`
	ErrorMessage  string         `gorm:"comment:错误信息" json:"error_message"`
	Success       bool           `gorm:"index;comment:是否成功" json:"success"`
	CreatedAt     time.Time      `gorm:"autoCreateTime;comment:创建时间" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime;comment:更新时间" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index;comment:删除时间(软删除)" json:"deleted_at"`
}

func (LLMLog) TableName() string { return "t_llm_logs" }
