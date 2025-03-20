// database.go の修正
package config

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// カスタムロガー
type customLogger struct {
	logger.Interface
}

func newCustomLogger() logger.Interface {
	return logger.New(
		log.New(log.Writer(), "[GORM] ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second, // 1秒以上のクエリを遅いと判断
			LogLevel:                  logger.Info, // Info レベル以上のログを表示
			IgnoreRecordNotFoundError: false,       // レコードが見つからないエラーも表示
			Colorful:                  true,        // カラフルなログ
		},
	)
}

// InitDB データベース接続を初期化
func InitDB(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.Username,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName)

	log.Printf("データベースに接続中: %s:%s/%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// GORM設定
	gormConfig := &gorm.Config{
		Logger: newCustomLogger(),
	}

	// データベースに接続
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, err
	}

	// 接続プールの設定
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// 接続プールの最大数を設定
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 接続テスト
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("データベース接続テストに失敗: %v", err)
	}

	log.Println("データベース接続に成功しました")

	return db, nil
}
