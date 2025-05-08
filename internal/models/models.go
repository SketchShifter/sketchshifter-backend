package models

import (
	"time"

	"gorm.io/gorm"
)

// User ユーザーモデル
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"`
	Name      string         `json:"name" gorm:"not null"`
	Nickname  string         `json:"nickname" gorm:"not null"`
	Bio       string         `json:"bio"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	Works    []Work    `json:"-"`
	Likes    []Like    `json:"-"`
	Comments []Comment `json:"-"`
}

// Tag タグモデル
type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`

	// リレーション
	Works []Work `json:"-" gorm:"many2many:work_tags;"`
}

// Work 作品モデル
type Work struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	Title             string         `json:"title" gorm:"not null"`
	Description       string         `json:"description"`
	FileData          []byte         `json:"-" gorm:"type:longblob"` // 互換性のために残す
	FileType          string         `json:"file_type"`
	FileName          string         `json:"file_name"`
	FileURL           string         `json:"file_url"`               // 追加
	FilePublicID      string         `json:"-"`                      // 追加
	ThumbnailData     []byte         `json:"-" gorm:"type:longblob"` // 互換性のために残す
	ThumbnailType     string         `json:"thumbnail_type"`
	ThumbnailURL      string         `json:"thumbnail_url"` // 追加
	ThumbnailPublicID string         `json:"-"`             // 追加
	CodeShared        bool           `json:"code_shared" gorm:"default:false"`
	CodeContent       string         `json:"code_content"`
	Views             int            `json:"views" gorm:"default:0"`
	UserID            *uint          `json:"user_id"`
	IsGuest           bool           `json:"is_guest" gorm:"default:false"`
	GuestNickname     string         `json:"guest_nickname"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	User     *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tags     []Tag     `json:"tags,omitempty" gorm:"many2many:work_tags;"`
	Likes    []Like    `json:"-"`
	Comments []Comment `json:"-"`

	// カウント (JSONレスポンス用)
	LikesCount    int64 `json:"likes_count" gorm:"-"`
	CommentsCount int64 `json:"comments_count" gorm:"-"`
}

// Like いいねモデル
type Like struct {
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	WorkID    uint      `json:"work_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`

	// リレーション
	User User `json:"-"`
	Work Work `json:"-"`
}

// Comment コメントモデル
type Comment struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Content       string         `json:"content" gorm:"not null"`
	WorkID        uint           `json:"work_id"`
	UserID        *uint          `json:"user_id"`
	IsGuest       bool           `json:"is_guest" gorm:"default:false"`
	GuestNickname string         `json:"guest_nickname"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Work Work  `json:"-"`
}

// ProcessingWork Processing作品モデル
type ProcessingWork struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	WorkID       uint           `json:"work_id" gorm:"not null;index"`
	OriginalName string         `json:"original_name"`
	PDEContent   string         `json:"pde_content" gorm:"type:text"`
	JSContent    string         `json:"js_content" gorm:"type:text"`
	CanvasID     string         `json:"canvas_id"`
	Status       string         `json:"status" gorm:"type:enum('pending','processing','processed','error');default:'pending'"`
	ErrorMessage string         `json:"error_message"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	Work Work `json:"work,omitempty" gorm:"foreignKey:WorkID"`
}

// WorkTag 作品とタグの中間テーブル
type WorkTag struct {
	WorkID uint `gorm:"primaryKey"`
	TagID  uint `gorm:"primaryKey"`
}

// TableName テーブル名指定
func (WorkTag) TableName() string {
	return "work_tags"
}
