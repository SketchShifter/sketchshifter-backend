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
	Projects []Project `json:"-" gorm:"foreignKey:OwnerID"`
}

// Tag タグモデル
type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`

	// リレーション
	Works []Work `json:"-" gorm:"many2many:work_tags;"`
}

// Work 作品モデル（ProcessingWorkを統合）
type Work struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	Title             string         `json:"title" gorm:"not null"`
	Description       string         `json:"description"`
	PDEContent        string         `json:"pde_content" gorm:"type:text"`
	JSContent         string         `json:"js_content" gorm:"type:text"`
	ThumbnailURL      string         `json:"thumbnail_url"`
	ThumbnailType     string         `json:"thumbnail_type"`
	ThumbnailPublicID string         `json:"-"`
	CodeShared        bool           `json:"code_shared" gorm:"default:false"`
	Views             int            `json:"views" gorm:"default:0"`
	UserID            uint           `json:"user_id" gorm:"not null"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	User     User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tags     []Tag     `json:"tags,omitempty" gorm:"many2many:work_tags;"`
	Likes    []Like    `json:"-"`
	Comments []Comment `json:"-"`
	Tasks    []Task    `json:"-" gorm:"many2many:task_works;"`

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
	ID        uint           `json:"id" gorm:"primaryKey"`
	Content   string         `json:"content" gorm:"not null"`
	WorkID    uint           `json:"work_id" gorm:"not null"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	User User `json:"user" gorm:"foreignKey:UserID"`
	Work Work `json:"-" gorm:"foreignKey:WorkID"`
}

// Project プロジェクトモデル
type Project struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Title          string         `json:"title" gorm:"not null"`
	Description    string         `json:"description"`
	InvitationCode string         `json:"invitation_code,omitempty" gorm:"uniqueIndex"`
	OwnerID        uint           `json:"owner_id" gorm:"not null"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	Owner   User   `json:"owner" gorm:"foreignKey:OwnerID"`
	Members []User `json:"members,omitempty" gorm:"many2many:project_members;"`
	Tasks   []Task `json:"tasks,omitempty"`
}

// ProjectMember プロジェクトメンバーモデル
type ProjectMember struct {
	ProjectID uint      `json:"project_id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"primaryKey"`
	IsOwner   bool      `json:"is_owner" gorm:"default:false"`
	JoinedAt  time.Time `json:"joined_at"`

	// リレーション
	Project Project `json:"-"`
	User    User    `json:"user"`
}

// Task タスクモデル
type Task struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Title       string         `json:"title" gorm:"not null"`
	Description string         `json:"description"`
	ProjectID   uint           `json:"project_id" gorm:"not null"`
	OrderIndex  int            `json:"order_index" gorm:"default:0"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// リレーション
	Project Project `json:"-" gorm:"foreignKey:ProjectID"`
	Works   []Work  `json:"works,omitempty" gorm:"many2many:task_works;"`
	Votes   []Vote  `json:"votes,omitempty"`
}

// TaskWork タスクと作品の中間テーブル
type TaskWork struct {
	TaskID    uint      `json:"task_id" gorm:"primaryKey"`
	WorkID    uint      `json:"work_id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"created_at"`
}

// Vote 投票モデル
type Vote struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"not null"`
	Description string     `json:"description"`
	TaskID      uint       `json:"task_id" gorm:"not null"`
	MultiSelect bool       `json:"multi_select" gorm:"default:false"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedBy   uint       `json:"created_by" gorm:"not null"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at"`

	// リレーション
	Task    Task         `json:"-" gorm:"foreignKey:TaskID"`
	Creator User         `json:"creator" gorm:"foreignKey:CreatedBy"`
	Options []VoteOption `json:"options,omitempty"`
}

// VoteOption 投票オプションモデル
type VoteOption struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	VoteID     uint      `json:"vote_id" gorm:"not null"`
	OptionText string    `json:"option_text" gorm:"not null"`
	WorkID     *uint     `json:"work_id"`
	CreatedAt  time.Time `json:"created_at"`

	// リレーション
	Vote Vote  `json:"-" gorm:"foreignKey:VoteID"`
	Work *Work `json:"work,omitempty" gorm:"foreignKey:WorkID"`

	// 投票数 (JSONレスポンス用)
	VoteCount int64 `json:"vote_count" gorm:"-"`
}

// VoteResponse 投票回答モデル
type VoteResponse struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	VoteID    uint      `json:"vote_id" gorm:"not null"`
	OptionID  uint      `json:"option_id" gorm:"not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`

	// リレーション
	Vote   Vote       `json:"-" gorm:"foreignKey:VoteID"`
	Option VoteOption `json:"-" gorm:"foreignKey:OptionID"`
	User   User       `json:"user" gorm:"foreignKey:UserID"`
}

// TableName テーブル名を指定
func (ProjectMember) TableName() string {
	return "project_members"
}

func (TaskWork) TableName() string {
	return "task_works"
}
