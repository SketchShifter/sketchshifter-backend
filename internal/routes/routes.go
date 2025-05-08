package routes

import (
	"log"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/controllers"
	"github.com/SketchShifter/sketchshifter_backend/internal/middlewares"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter ルーターを設定
func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	// Ginルーターを作成
	r := gin.Default()

	// ミドルウェアを設定
	r.Use(middlewares.ErrorMiddleware())
	r.Use(middlewares.CORSMiddleware())

	// リポジトリを作成
	userRepo := repository.NewUserRepository(db)
	workRepo := repository.NewWorkRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	processingRepo := repository.NewProcessingRepository(db)

	// Lambda関連のサービスを作成
	lambdaService := services.NewLambdaService(cfg, processingRepo)

	// Cloudinaryサービスを作成
	cloudinaryService, err := services.NewCloudinaryService(cfg)
	if err != nil {
		log.Fatalf("Cloudinaryサービスの初期化に失敗しました: %v", err)
	}

	// サービスを作成
	authService := services.NewAuthService(userRepo, cfg)
	workService := services.NewWorkService(
		workRepo,
		tagRepo,
		processingRepo,
		lambdaService,
		cloudinaryService, // 追加
	)
	tagService := services.NewTagService(tagRepo)
	commentService := services.NewCommentService(commentRepo, workRepo)
	userService := services.NewUserService(userRepo, workRepo)

	// コントローラーを作成
	authController := controllers.NewAuthController(authService)
	workController := controllers.NewWorkController(workService)
	tagController := controllers.NewTagController(tagService)
	commentController := controllers.NewCommentController(commentService)
	userController := controllers.NewUserController(userService)
	healthController := controllers.NewHealthController()
	lambdaController := controllers.NewLambdaController(lambdaService)

	// 認証ミドルウェア
	authMiddleware := middlewares.AuthMiddleware(authService)
	optionalAuthMiddleware := middlewares.OptionalAuthMiddleware(authService)

	// APIグループを作成
	api := r.Group("/api/v1")
	{
		// ヘルスチェックルート（認証不要）
		api.GET("/health", healthController.Check)

		// 認証ルート
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.GET("/me", authMiddleware, authController.GetMe)
			auth.POST("/change-password", authMiddleware, authController.ChangePassword)
		}

		// 作品ルート
		works := api.Group("/works")
		{
			// 認証不要
			works.GET("", optionalAuthMiddleware, workController.List)
			works.GET("/:id", optionalAuthMiddleware, workController.GetByID)
			works.GET("/:id/file", workController.GetFile)
			works.GET("/:id/thumbnail", workController.GetThumbnail)
			works.POST("/preview", workController.CreatePreview)

			// コメント関連
			works.GET("/:id/comments", commentController.List)
			works.POST("/:id/comments", optionalAuthMiddleware, commentController.Create)

			// 認証が必要
			works.GET("/:id/liked", authMiddleware, workController.HasLiked)
			works.POST("", authMiddleware, workController.Create)
			works.PUT("/:id", authMiddleware, workController.Update)
			works.DELETE("/:id", authMiddleware, workController.Delete)
			works.POST("/:id/like", authMiddleware, workController.AddLike)
			works.DELETE("/:id/like", authMiddleware, workController.RemoveLike)
		}

		// Lambda関連ルート（デバッグ・管理用）
		lambda := api.Group("/lambda")
		{
			// 手動変換起動（デバッグ用）
			lambda.POST("/process/:id", authMiddleware, lambdaController.ProcessPDE)
		}

		// コメントルート
		comments := api.Group("/comments")
		{
			comments.PUT("/:id", authMiddleware, commentController.Update)
			comments.DELETE("/:id", authMiddleware, commentController.Delete)
		}

		// タグルート
		api.GET("/tags", tagController.List)

		// ユーザールート
		users := api.Group("/users")
		{
			users.GET("/:id", userController.GetByID)
			users.GET("/:id/works", userController.GetUserWorks)
			users.GET("/favorites", authMiddleware, userController.GetUserFavorites)
			users.GET("/me", authMiddleware, userController.GetMe)
			users.PUT("/profile", authMiddleware, userController.UpdateProfile)
			users.GET("/my-works", authMiddleware, userController.GetMyWorks)
		}
	}

	return r
}
