package entity

import "gorm.io/gorm"

type LLMLog struct {
	gorm.Model
	URLID         uint    `gorm:"index" json:"url_id"`
	RequestType   string  `gorm:"index" json:"request_type"` // chat | embedding
	Provider      string  `json:"provider"`
	ModelName     string  `gorm:"column:model" json:"model"`
	PromptKey     string  `json:"prompt_key"`
	InputContent  string  `gorm:"type:mediumtext" json:"input_content"`
	OutputContent string  `gorm:"type:mediumtext" json:"output_content"`
	InputTokens   int     `json:"input_tokens"`
	OutputTokens  int     `json:"output_tokens"`
	TotalTokens   int     `json:"total_tokens"`
	LatencyMs     int64   `json:"latency_ms"`
	TokensPerSec  float64 `json:"tokens_per_sec"`
	StatusCode    int     `json:"status_code"`
	ErrorMessage  string  `json:"error_message"`
	Success       bool    `gorm:"index" json:"success"`
}

func (LLMLog) TableName() string { return "t_llm_logs" }
