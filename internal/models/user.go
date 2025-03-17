package models

import (
	"time"
)

// User はユーザーモデル
type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"unique;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	Name         string    `json:"name" gorm:"not null"`
	Nickname     string    `json:"nickname" gorm:"not null"`
	AvatarURL    string    `json:"avatar_url"`
	Bio          string    `json:"bio"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
