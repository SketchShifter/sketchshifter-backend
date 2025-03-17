package models

import (
	"time"
)

// Work は作品モデル
type Work struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        *uint     `json:"user_id"`
	Title         string    `json:"title" gorm:"not null"`
	Description   string    `json:"description"`
	FileURL       string    `json:"file_url" gorm:"not null"`
	ThumbnailURL  string    `json:"thumbnail_url"`
	CodeShared    bool      `json:"code_shared" gorm:"default:false"`
	CodeContent   string    `json:"code_content"`
	Views         int       `json:"views" gorm:"default:0"`
	IsGuest       bool      `json:"is_guest" gorm:"default:false"`
	GuestNickname string    `json:"guest_nickname"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Tags          []Tag     `json:"tags" gorm:"many2many:work_tags;"`
}
