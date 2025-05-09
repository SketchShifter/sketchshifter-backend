package routes

import (
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
	projectRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	voteRepo := repository.NewVoteRepository(db)

	// Cloudinaryサービスを作成
	cloudinaryService, err := services.NewCloudinaryService(cfg)
	if err != nil {
		panic("Cloudinaryサービスの初期化に失敗しました: " + err.Error())
	}

	// Lambdaサービスを作成
	lambdaService := services.NewLambdaService(cfg)

	// サービスを作成
	authService := services.NewAuthService(userRepo, cfg)
	workService := services.NewWorkService(workRepo, tagRepo, cloudinaryService, lambdaService)
	tagService := services.NewTagService(tagRepo)
	commentService := services.NewCommentService(commentRepo, workRepo)
	userService := services.NewUserService(userRepo, workRepo)
	projectService := services.NewProjectService(projectRepo, taskRepo)
	taskService := services.NewTaskService(taskRepo, projectRepo, workRepo)
	voteService := services.NewVoteService(voteRepo, taskRepo, projectRepo, workRepo)

	// コントローラーを作成
	authController := controllers.NewAuthController(authService)
	workController := controllers.NewWorkController(workService)
	tagController := controllers.NewTagController(tagService)
	commentController := controllers.NewCommentController(commentService)
	userController := controllers.NewUserController(userService)
	healthController := controllers.NewHealthController()
	projectController := controllers.NewProjectController(projectService)
	taskController := controllers.NewTaskController(taskService)
	voteController := controllers.NewVoteController(voteService)

	// 認証ミドルウェア
	authMiddleware := middlewares.AuthMiddleware(authService)

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
			works.GET("", workController.List)
			works.GET("/:id", workController.GetByID)

			// コメント関連
			works.GET("/:id/comments", commentController.List)
			works.POST("/:id/comments", authMiddleware, commentController.Create)

			// 認証が必要
			works.GET("/:id/liked", authMiddleware, workController.HasLiked)
			works.POST("", authMiddleware, workController.Create)
			works.PUT("/:id", authMiddleware, workController.Update)
			works.DELETE("/:id", authMiddleware, workController.Delete)
			works.POST("/:id/like", authMiddleware, workController.AddLike)
			works.DELETE("/:id/like", authMiddleware, workController.RemoveLike)
		}

		// コメントルート
		comments := api.Group("/comments").Use(authMiddleware)
		{
			comments.PUT("/:id", commentController.Update)
			comments.DELETE("/:id", commentController.Delete)
		}

		// タグルート
		api.GET("/tags", tagController.List)

		// ユーザールート
		users := api.Group("/users")
		{
			users.GET("/:id", userController.GetByID)
			users.GET("/:userID/works", workController.GetUserWorks)
			users.GET("/me", authMiddleware, userController.GetMe)
			users.PUT("/profile", authMiddleware, userController.UpdateProfile)
		}

		// プロジェクトルート
		projects := api.Group("/projects").Use(authMiddleware)
		{
			projects.GET("", projectController.List)
			projects.POST("", projectController.Create)
			projects.GET("/my", projectController.GetUserProjects)
			projects.POST("/join", projectController.JoinProject)
			projects.GET("/:id", projectController.GetByID)
			projects.PUT("/:id", projectController.Update)
			projects.DELETE("/:id", projectController.Delete)
			projects.GET("/:id/members", projectController.GetMembers)
			projects.DELETE("/:id/members/:memberID", projectController.RemoveMember)
			projects.POST("/:id/invitation-code", projectController.GenerateInvitationCode)
		}

		// タスクルート
		tasks := api.Group("/tasks").Use(authMiddleware)
		{
			tasks.POST("", taskController.Create)
			tasks.GET("/:id", taskController.GetByID)
			tasks.PUT("/:id", taskController.Update)
			tasks.DELETE("/:id", taskController.Delete)
			tasks.GET("/project/:projectID", taskController.ListByProject)
			tasks.POST("/:id/works", taskController.AddWork)
			tasks.DELETE("/:id/works/:workID", taskController.RemoveWork)
			tasks.GET("/:id/works", taskController.GetWorks)
			tasks.PUT("/orders", taskController.UpdateOrders)
		}

		// 投票ルート
		votes := api.Group("/votes").Use(authMiddleware)
		{
			votes.POST("", voteController.Create)
			votes.GET("/:id", voteController.GetByID)
			votes.PUT("/:id", voteController.Update)
			votes.DELETE("/:id", voteController.Delete)
			votes.GET("/task/:taskID", voteController.ListByTask)
			votes.POST("/:id/options", voteController.AddOption)
			votes.DELETE("/:id/options/:optionID", voteController.DeleteOption)
			votes.POST("/:id/vote", voteController.Vote)
			votes.DELETE("/:id/vote/:optionID", voteController.RemoveVote)
			votes.GET("/:id/user-votes", voteController.GetUserVotes)
			votes.POST("/:id/close", voteController.CloseVote)
		}
	}

	return r
}
