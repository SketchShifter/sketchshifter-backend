package main

import (
	"log"
	"os"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	// ログ設定を変更
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("サーバーを起動しています...")

	// 設定をロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// コマンドライン引数をチェック
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// マイグレーションモードで実行
		handleMigration(cfg, os.Args[2:])
		return
	}

	// Gin モードの設定（環境変数が設定されていない場合はデバッグモード）
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(ginMode)
	}

	// カスタムログフォーマットを設定
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		log.Printf("エンドポイント登録: %s %s -> %s (%d handlers)\n", httpMethod, absolutePath, handlerName, nuHandlers)
	}

	// データベース接続
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}

	// ルーターをセットアップ
	router := routes.SetupRouter(cfg, db)

	// サーバー起動
	log.Printf("サーバーを開始しています... PORT: %s, MODE: %s", cfg.Server.Port, gin.Mode())
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}

// マイグレーション処理を実行
func handleMigration(cfg *config.Config, args []string) {
	if len(args) == 0 {
		log.Fatal("使用方法: app migrate [up|down]")
	}

	command := args[0]

	// データベース接続
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}

	switch command {
	case "up":
		// マイグレーションを実行
		log.Println("マイグレーションを実行中...")
		err = db.AutoMigrate(
			&models.User{},
			&models.Tag{},
			&models.Work{},
			&models.Like{},
			&models.Comment{},
			&models.Project{},
			&models.ProjectMember{},
			&models.Task{},
			&models.TaskWork{},
			&models.Vote{},
			&models.VoteOption{},
			&models.VoteResponse{},
		)
		if err != nil {
			log.Fatalf("マイグレーションに失敗しました: %v", err)
		}
		log.Println("マイグレーションが成功しました")

	case "down":
		// テーブルを削除（逆順）
		log.Println("マイグレーションをロールバック中...")
		err = db.Migrator().DropTable(
			&models.VoteResponse{},
			&models.VoteOption{},
			&models.Vote{},
			&models.TaskWork{},
			&models.Task{},
			&models.ProjectMember{},
			&models.Project{},
			&models.Comment{},
			&models.Like{},
			"work_tags",
			&models.Work{},
			&models.Tag{},
			&models.User{},
		)
		if err != nil {
			log.Fatalf("テーブル削除に失敗しました: %v", err)
		}
		log.Println("テーブルの削除が成功しました")

	default:
		log.Fatalf("不明なコマンドです: %s", command)
	}
}
