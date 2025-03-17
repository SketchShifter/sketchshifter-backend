package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// .envファイルを読み込み
	godotenv.Load()

	// ポート設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // デフォルトポート
	}

	// Ginエンジンの初期化
	r := gin.Default()

	// CORSの設定
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API エンドポイント
	api := r.Group("/api")
	{
		// 作品エンドポイント
		api.GET("/works", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"works": []gin.H{
					{
						"id":            1,
						"title":         "Particle System",
						"description":   "An interactive particle system that responds to mouse movements",
						"file_url":      "/uploads/works/particle_system.pde",
						"thumbnail_url": "https://via.placeholder.com/300x200",
						"code_shared":   true,
						"code_content":  "void setup() {\n  size(800, 600);\n  background(0);\n}\n\nvoid draw() {\n  // Particle system code\n}",
						"views":         120,
						"is_guest":      false,
						"created_at":    "2024-02-26T12:00:00Z",
						"updated_at":    "2024-02-26T12:00:00Z",
						"user": gin.H{
							"id":         1,
							"nickname":   "johndoe",
							"avatar_url": "https://via.placeholder.com/150",
						},
						"tags": []gin.H{
							{"id": 1, "name": "animation"},
							{"id": 4, "name": "particles"},
						},
						"likes_count":    12,
						"comments_count": 5,
					},
					{
						"id":            2,
						"title":         "Generative Landscape",
						"description":   "A procedurally generated landscape that changes over time",
						"file_url":      "/uploads/works/generative_landscape.pde",
						"thumbnail_url": "https://via.placeholder.com/300x200",
						"code_shared":   true,
						"code_content":  "void setup() {\n  size(800, 600, P3D);\n  background(0);\n}\n\nvoid draw() {\n  // Landscape generation code\n}",
						"views":         85,
						"is_guest":      false,
						"created_at":    "2024-03-05T14:30:00Z",
						"updated_at":    "2024-03-05T14:30:00Z",
						"user": gin.H{
							"id":         2,
							"nickname":   "janesmith",
							"avatar_url": "https://via.placeholder.com/150",
						},
						"tags": []gin.H{
							{"id": 3, "name": "generative"},
							{"id": 5, "name": "3D"},
						},
						"likes_count":    8,
						"comments_count": 3,
					},
				},
				"total": 2,
				"pages": 1,
				"page":  1,
			})
		})

		// 作品詳細
		api.GET("/works/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(200, gin.H{
				"id":            id,
				"title":         "Particle System",
				"description":   "An interactive particle system that responds to mouse movements",
				"file_url":      "/uploads/works/particle_system.pde",
				"thumbnail_url": "https://via.placeholder.com/300x200",
				"code_shared":   true,
				"code_content":  "void setup() {\n  size(800, 600);\n  background(0);\n}\n\nvoid draw() {\n  // Particle system code\n}",
				"views":         120,
				"is_guest":      false,
				"created_at":    "2024-02-26T12:00:00Z",
				"updated_at":    "2024-02-26T12:00:00Z",
				"user": gin.H{
					"id":         1,
					"nickname":   "johndoe",
					"avatar_url": "https://via.placeholder.com/150",
				},
				"tags": []gin.H{
					{"id": 1, "name": "animation"},
					{"id": 4, "name": "particles"},
				},
				"likes_count":    12,
				"comments_count": 5,
			})
		})

		// コメント取得
		api.GET("/works/:id/comments", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"comments": []gin.H{
					{
						"id":      1,
						"content": "Amazing work! How did you achieve that particle effect?",
						"user": gin.H{
							"id":         2,
							"nickname":   "janesmith",
							"avatar_url": "https://via.placeholder.com/150",
						},
						"is_guest":   false,
						"created_at": "2024-02-28T09:15:00Z",
					},
					{
						"id":      2,
						"content": "Thanks! I used a vector field to control the particle movement.",
						"user": gin.H{
							"id":         1,
							"nickname":   "johndoe",
							"avatar_url": "https://via.placeholder.com/150",
						},
						"is_guest":   false,
						"created_at": "2024-02-28T10:30:00Z",
					},
				},
				"total": 2,
				"pages": 1,
				"page":  1,
			})
		})

		// タグ取得
		api.GET("/tags", func(c *gin.Context) {
			c.JSON(200, []gin.H{
				{"id": 1, "name": "animation"},
				{"id": 2, "name": "interactive"},
				{"id": 3, "name": "generative"},
				{"id": 4, "name": "particles"},
				{"id": 5, "name": "3D"},
			})
		})

		// 認証エンドポイント
		auth := api.Group("/auth")
		{
			auth.POST("/login", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJleHAiOjE2MTQwMjIwMDAsImlhdCI6MTYxNDAxODQwMH0.mock_token",
					"user": gin.H{
						"id":         1,
						"email":      "john@example.com",
						"name":       "John Doe",
						"nickname":   "johndoe",
						"avatar_url": "https://via.placeholder.com/150",
						"bio":        "Processing enthusiast and creative coder",
					},
				})
			})

			auth.POST("/register", func(c *gin.Context) {
				c.JSON(201, gin.H{
					"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjozLCJleHAiOjE2MTQwMjIwMDAsImlhdCI6MTYxNDAxODQwMH0.mock_token",
					"user": gin.H{
						"id":       3,
						"email":    c.PostForm("email"),
						"name":     c.PostForm("name"),
						"nickname": c.PostForm("nickname"),
					},
				})
			})
		}
	}

	// サーバー起動
	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
