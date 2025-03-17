package mock

import (
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"time"
)

// モックユーザー
var Users = []models.User{
	{
		ID:          1,
		Email:       "john@example.com",
		PasswordHash: "$2a$10$eDxe8U2bkJFVt1C1vfVJJePg8GVyp5eZZP/EaQ/2e8LqNUvpBtqOW", // "password"
		Name:        "John Doe",
		Nickname:    "johndoe",
		AvatarURL:   "https://via.placeholder.com/150",
		Bio:         "Processing enthusiast and creative coder",
		CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:   time.Now().Add(-10 * 24 * time.Hour),
	},
	{
		ID:          2,
		Email:       "jane@example.com",
		PasswordHash: "$2a$10$eDxe8U2bkJFVt1C1vfVJJePg8GVyp5eZZP/EaQ/2e8LqNUvpBtqOW", // "password"
		Name:        "Jane Smith",
		Nickname:    "janesmith",
		AvatarURL:   "https://via.placeholder.com/150",
		Bio:         "Digital artist and generative art creator",
		CreatedAt:   time.Now().Add(-25 * 24 * time.Hour),
		UpdatedAt:   time.Now().Add(-5 * 24 * time.Hour),
	},
}

// モックタグ
var Tags = []models.Tag{
	{ID: 1, Name: "animation", CreatedAt: time.Now().Add(-40 * 24 * time.Hour)},
	{ID: 2, Name: "interactive", CreatedAt: time.Now().Add(-38 * 24 * time.Hour)},
	{ID: 3, Name: "generative", CreatedAt: time.Now().Add(-35 * 24 * time.Hour)},
	{ID: 4, Name: "particles", CreatedAt: time.Now().Add(-32 * 24 * time.Hour)},
	{ID: 5, Name: "3D", CreatedAt: time.Now().Add(-30 * 24 * time.Hour)},
}

// モック作品
var Works = []models.Work{
	{
		ID:           1,
		UserID:       &Users[0].ID,
		Title:        "Particle System",
		Description:  "An interactive particle system that responds to mouse movements",
		FileURL:      "/uploads/works/particle_system.pde",
		ThumbnailURL: "https://via.placeholder.com/300x200",
		CodeShared:   true,
		CodeContent:  "void setup() {\n  size(800, 600);\n  background(0);\n}\n\nvoid draw() {\n  // Particle system code\n}",
		Views:        120,
		IsGuest:      false,
		CreatedAt:    time.Now().Add(-20 * 24 * time.Hour),
		UpdatedAt:    time.Now().Add(-20 * 24 * time.Hour),
		Tags:         []models.Tag{Tags[0], Tags[3]},
	},
	{
		ID:           2,
		UserID:       &Users[1].ID,
		Title:        "Generative Landscape",
		Description:  "A procedurally generated landscape that changes over time",
		FileURL:      "/uploads/works/generative_landscape.pde",
		ThumbnailURL: "https://via.placeholder.com/300x200",
		CodeShared:   true,
		CodeContent:  "void setup() {\n  size(800, 600, P3D);\n  background(0);\n}\n\nvoid draw() {\n  // Landscape generation code\n}",
		Views:        85,
		IsGuest:      false,
		CreatedAt:    time.Now().Add(-15 * 24 * time.Hour),
		UpdatedAt:    time.Now().Add(-15 * 24 * time.Hour),
		Tags:         []models.Tag{Tags[2], Tags[4]},
	},
	{
		ID:           3,
		UserID:       &Users[0].ID,
		Title:        "Interactive Pattern",
		Description:  "Click and drag to create interesting patterns",
		FileURL:      "/uploads/works/interactive_pattern.pde",
		ThumbnailURL: "https://via.placeholder.com/300x200",
		CodeShared:   false,
		Views:        42,
		IsGuest:      false,
		CreatedAt:    time.Now().Add(-10 * 24 * time.Hour),
		UpdatedAt:    time.Now().Add(-10 * 24 * time.Hour),
		Tags:         []models.Tag{Tags[1]},
	},
	{
		ID:            4,
		Title:         "Guest Submission",
		Description:   "A simple animation created by a guest user",
		FileURL:       "/uploads/works/guest_animation.pde",
		ThumbnailURL:  "https://via.placeholder.com/300x200",
		CodeShared:    true,
		CodeContent:   "void setup() {\n  size(500, 500);\n  background(0);\n}\n\nvoid draw() {\n  // Simple animation code\n}",
		Views:         12,
		IsGuest:       true,
		GuestNickname: "anonymous_artist",
		CreatedAt:     time.Now().Add(-5 * 24 * time.Hour),
		UpdatedAt:     time.Now().Add(-5 * 24 * time.Hour),
		Tags:          []models.Tag{Tags[0]},
	},
}

// モックコメント
var Comments = []models.Comment{
	{
		ID:        1,
		WorkID:    1,
		UserID:    &Users[1].ID,
		Content:   "Amazing work! How did you achieve that particle effect?",
		IsGuest:   false,
		CreatedAt: time.Now().Add(-18 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-18 * 24 * time.Hour),
	},
	{
		ID:        2,
		WorkID:    1,
		UserID:    &Users[0].ID,
		Content:   "Thanks! I used a vector field to control the particle movement.",
		IsGuest:   false,
		CreatedAt: time.Now().Add(-17 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-17 * 24 * time.Hour),
	},
	{
		ID:            3,
		WorkID:        2,
		Content:       "This is very beautiful, I love the colors!",
		IsGuest:       true,
		GuestNickname: "art_lover",
		CreatedAt:     time.Now().Add(-12 * 24 * time.Hour),
		UpdatedAt:     time.Now().Add(-12 * 24 * time.Hour),
	},
	{
		ID:        4,
		WorkID:    3,
		UserID:    &Users[1].ID,
		Content:   "Would you consider sharing the code for this one?",
		IsGuest:   false,
		CreatedAt: time.Now().Add(-8 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-8 * 24 * time.Hour),
	},
}

// モックいいね
var Likes = []models.Like{
	{ID: 1, WorkID: 1, UserID: Users[1].ID, CreatedAt: time.Now().Add(-19 * 24 * time.Hour)},
	{ID: 2, WorkID: 2, UserID: Users[0].ID, CreatedAt: time.Now().Add(-14 * 24 * time.Hour)},
	{ID: 3, WorkID: 3, UserID: Users[1].ID, CreatedAt: time.Now().Add(-9 * 24 * time.Hour)},
}

// モックお気に入り
var Favorites = []models.Favorite{
	{ID: 1, WorkID: 1, UserID: Users[1].ID, CreatedAt: time.Now().Add(-18 * 24 * time.Hour)},
	{ID: 2, WorkID: 2, UserID: Users[0].ID, CreatedAt: time.Now().Add(-13 * 24 * time.Hour)},
}
