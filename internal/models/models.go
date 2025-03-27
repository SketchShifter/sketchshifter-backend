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
	AvatarURL string         `json:"avatar_url"`
	Bio       string         `json:"bio"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	ExternalAccounts []ExternalAccount `json:"-"`
	Works            []Work            `json:"-"`
	Likes            []Like            `json:"-"`
	Comments         []Comment         `json:"-"`
}

// ExternalAccount 外部認証アカウント
type ExternalAccount struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	CreatedAt  time.Time `json:"created_at"`

	// リレーション
	User User `json:"-"`
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
	ID            uint           `json:"id" gorm:"primaryKey"`
	Title         string         `json:"title" gorm:"not null"`
	Description   string         `json:"description"`
	FileURL       string         `json:"file_url" gorm:"not null"`
	ThumbnailURL  string         `json:"thumbnail_url"`
	CodeShared    bool           `json:"code_shared" gorm:"default:false"`
	CodeContent   string         `json:"code_content"`
	Views         int            `json:"views" gorm:"default:0"`
	UserID        *uint          `json:"user_id"`
	IsGuest       bool           `json:"is_guest" gorm:"default:false"`
	GuestNickname string         `json:"guest_nickname"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

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
	FileName     string         `json:"file_name" gorm:"not null"`
	OriginalName string         `json:"original_name"`
	PDEContent   string         `json:"pde_content" gorm:"type:text"` // PDEファイルの内容を直接保存
	PDEPath      string         `json:"pde_path"`                     // R2へのパスは省略可能に
	JSPath       string         `json:"js_path"`
	CanvasID     string         `json:"canvas_id"`
	Status       string         `json:"status" gorm:"type:enum('pending','processing','processed','error');default:'pending'"`
	ErrorMessage string         `json:"error_message"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	Work Work `json:"work,omitempty" gorm:"foreignKey:WorkID"`
}

// Image 画像モデル
type Image struct {
	ID               uint           `json:"id" gorm:"primaryKey"`
	WorkID           *uint          `json:"work_id" gorm:"index"`
	FileName         string         `json:"file_name" gorm:"not null"`
	OriginalPath     string         `json:"original_path" gorm:"not null"`
	WebpPath         string         `json:"webp_path"`
	Status           string         `json:"status" gorm:"type:enum('pending','processing','processed','error');default:'pending'"`
	ErrorMessage     string         `json:"error_message"`
	OriginalSize     int64          `json:"original_size" gorm:"default:0"`
	WebpSize         int64          `json:"webp_size" gorm:"default:0"`
	CompressionRatio float64        `json:"compression_ratio" gorm:"default:0"`
	Width            int            `json:"width" gorm:"default:0"`
	Height           int            `json:"height" gorm:"default:0"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	Work *Work `json:"work,omitempty" gorm:"foreignKey:WorkID"`
}

// TableName テーブル名を指定
func (ProcessingWork) TableName() string {
	return "processing_works"
}

// TableName テーブル名を指定
func (User) TableName() string {
	return "users"
}

func (ExternalAccount) TableName() string {
	return "external_accounts"
}

func (Tag) TableName() string {
	return "tags"
}

func (Work) TableName() string {
	return "works"
}

func (Like) TableName() string {
	return "likes"
}

func (Comment) TableName() string {
	return "comments"
}
