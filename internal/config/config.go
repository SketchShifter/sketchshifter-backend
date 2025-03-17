package config

import (
	"os"
	"github.com/joho/godotenv"
)

// Config はアプリケーション設定を保持する構造体
type Config struct {
	Port         string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	JWTSecret    string
	StoragePath  string
}

// Load は環境変数から設定を読み込む
func Load() (*Config, error) {
	// .envファイルの読み込み（存在する場合）
	godotenv.Load()

	config := &Config{
		Port:         getEnv("PORT", "8080"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgres"),
		DBName:       getEnv("DB_NAME", "sketchshifter"),
		JWTSecret:    getEnv("JWT_SECRET", "your_jwt_secret_key"),
		StoragePath:  getEnv("STORAGE_PATH", "./uploads"),
	}

	return config, nil
}

// getEnv は環境変数を取得し、設定されていない場合はデフォルト値を返す
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
