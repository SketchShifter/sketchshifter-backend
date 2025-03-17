package models

import (
	"time"
)

// Comment はコメントモデル
type Comment struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	WorkID        uint      `json:"work_id" gorm:"not null"`
	UserID        *uint     `json:"user_id"`
	Content       string    `json:"content" gorm:"not null"`
	IsGuest       bool      `json:"is_guest" gorm:"default:false"`
	GuestNickname string    `json:"guest_nickname"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
