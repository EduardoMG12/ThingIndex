package models

import "time"

type UserProviderConfig struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	UserID        uint      `gorm:"not null" json:"user_id"`
	ProviderType  string    `gorm:"size:50;not null;uniqueIndex:idx_user_provider" json:"provider_type"` // e.g., 'OPENAI', 'AWS_S3'
	EncryptedData string    `gorm:"type:text;not null" json:"encrypted_data"`                             // JSON encrypted with AES
	IsEnabled     bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
