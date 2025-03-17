package models

import (
	"time"
)

// Tag はタグモデル
type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"unique;not null"`
	CreatedAt time.Time `json:"created_at"`
}
