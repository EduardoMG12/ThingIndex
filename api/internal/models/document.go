package models

import (
	"time"

	"gorm.io/gorm"
)

type DocumentCategory struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	UserID      uint           `gorm:"not null" json:"user_id"`
	Name        string         `gorm:"size:255;uniqueIndex:idx_user_category;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type Document struct {
	ID         uint              `gorm:"primarykey" json:"id"`
	UserID     uint              `gorm:"not null" json:"user_id"`
	Title      string            `gorm:"size:255;not null" json:"title"`
	FileURL    string            `gorm:"size:512;not null" json:"file_url"`
	Size       int64             `json:"size"`
	MimeType   string            `gorm:"size:100" json:"mime_type"`
	Status     string            `gorm:"size:50;default:'PENDING'" json:"status"` // E.g., PENDING, MAPPED, FAILED
	Collection string            `gorm:"size:255;index" json:"collection,omitempty"`
	CategoryID *uint             `json:"category_id,omitempty"` // FK to DocumentCategory
	Category   *DocumentCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	DeletedAt  gorm.DeletedAt    `gorm:"index" json:"-"`
}
