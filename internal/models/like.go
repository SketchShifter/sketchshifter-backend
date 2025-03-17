package models

import (
	"time"
)

// Like はいいねモデル
type Like struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	WorkID    uint      `json:"work_id" gorm:"not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}
