package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                  uint           `gorm:"primarykey" json:"id"`
	Name                string         `gorm:"size:255;not null" json:"name"`
	Email               string         `gorm:"size:255;unique;not null" json:"email"`
	Password            string         `gorm:"size:255;not null" json:"-"`
	ResetToken          string         `gorm:"size:255" json:"-"`
	ResetTokenExpiresAt *time.Time     `json:"-"`
	SessionInvalidAfter *time.Time     `json:"-"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}
