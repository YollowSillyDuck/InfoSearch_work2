package models

import (
	"time"

	"gorm.io/gorm"
)

type Document struct {
	gorm.Model
	Title       string     `gorm:"type:varchar(255);not null" json:"title"`
	Summary     string     `gorm:"type:text" json:"summary"`
	Content     string     `gorm:"type:longtext;not null" json:"content"`
	URL         string     `gorm:"type:varchar(255)" json:"url"`
	Author      string     `gorm:"type:varchar(128)" json:"author"`
	Source      string     `gorm:"type:varchar(128)" json:"source"`
	Status      string     `gorm:"type:varchar(32);default:'published'" json:"status"`
	PublishedAt *time.Time `json:"published_at"`
	Tags        []Tag      `gorm:"many2many:document_tags;" json:"tags,omitempty"`
	WordCount   int        `gorm:"default:0" json:"word_count"`
}

func (Document) TableName() string {
	return "documents"
}

type Tag struct {
	gorm.Model
	Name      string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	Documents []Document `gorm:"many2many:document_tags;" json:"-"`
}

func (Tag) TableName() string {
	return "tags"
}

type DocumentTag struct {
	DocumentID uint `gorm:"primaryKey" json:"document_id"`
	TagID      uint `gorm:"primaryKey" json:"tag_id"`
}

func (DocumentTag) TableName() string {
	return "document_tags"
}
