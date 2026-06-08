package models

import (
	"time"

	"gorm.io/gorm"
)

type SearchRecord struct {
	gorm.Model
	Query       string    `gorm:"type:varchar(256);not null" json:"query"`
	UserID      uint      `json:"user_id"`
	UserIP      string    `gorm:"type:varchar(64)" json:"user_ip"`
	ResultCount int       `gorm:"default:0" json:"result_count"`
	DurationMs  int64     `gorm:"default:0" json:"duration_ms"`
	Filters     string    `gorm:"type:text" json:"filters"`
	SearchedAt  time.Time `gorm:"autoCreateTime" json:"searched_at"`
}

func (SearchRecord) TableName() string {
	return "search_records"
}

type SearchRequest struct {
	Query    string    `form:"query" json:"query" binding:"required"`
	Page     int       `form:"page,default=1" json:"page"`
	PageSize int       `form:"page_size,default=20" json:"page_size"`
	TagIDs   []uint    `form:"tag_ids[]" json:"tag_ids"`
	Sort     string    `form:"sort" json:"sort"`
	StartAt  time.Time `form:"start_at" json:"start_at"`
	EndAt    time.Time `form:"end_at" json:"end_at"`
}
// models/search.go 追加以下代码

type SearchEvaluation struct {
	gorm.Model
	Query      string `gorm:"type:varchar(256);not null" json:"query" binding:"required"`
	DocumentID uint   `json:"document_id" binding:"required"`
	IsRelevant bool   `json:"is_relevant"` // true为相关(准)，false为不相关(不准)
	UserIP     string `gorm:"type:varchar(64)" json:"user_ip"`
}

func (SearchEvaluation) TableName() string {
	return "search_evaluations"
}