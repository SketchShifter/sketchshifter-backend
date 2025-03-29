package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config アプリケーション設定
type Config struct {
	Server           ServerConfig
	Database         DatabaseConfig
	Auth             AuthConfig
	Storage          StorageConfig
	AWS              AWSConfig
	CloudflareWorker CloudflareWorkerConfig // 追加: Cloudflare Worker設定
}

// ServerConfig サーバー設定
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	APIBaseURL   string // APIのベースURL（Lambda関数のコールバック用）
}

// DatabaseConfig データベース設定
type DatabaseConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
}

// AuthConfig 認証設定
type AuthConfig struct {
	JWTSecret          string
	TokenExpiry        time.Duration
	GoogleClientID     string
	GoogleClientSecret string
	GithubClientID     string
	GithubClientSecret string
}

// StorageConfig ストレージ設定
type StorageConfig struct {
	UploadDir     string
	MaxUploadSize int64
	AllowedTypes  []string
}

// AWSConfig AWS設定
type AWSConfig struct {
	Region         string
	LambdaEndpoint string // PDE変換Lambda関数のエンドポイント
}

// CloudflareWorkerConfig Cloudflare Worker設定
type CloudflareWorkerConfig struct {
	URL     string // Cloudflare Workerのエンドポイント
	APIKey  string // Cloudflare WorkerのAPIキー
	Bucket  string // Cloudflare R2のバケット名
	Enabled bool   // 有効かどうか
}

// Load 環境変数から設定をロード
func Load() (*Config, error) {
	// .env ファイルをロード (存在すれば)
	_ = godotenv.Load()

	// デフォルト値を設定
	config := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  time.Duration(getEnvAsInt("SERVER_READ_TIMEOUT", 10)) * time.Second,
			WriteTimeout: time.Duration(getEnvAsInt("SERVER_WRITE_TIMEOUT", 10)) * time.Second,
			APIBaseURL:   getEnv("API_BASE_URL", "https://api.serendicode-sub.click"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			Username: getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "processing_platform"),
		},
		Auth: AuthConfig{
			JWTSecret:          getEnv("JWT_SECRET", "your-secret-key"),
			TokenExpiry:        time.Duration(getEnvAsInt("TOKEN_EXPIRY", 24)) * time.Hour,
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			GithubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
			GithubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		},
		Storage: StorageConfig{
			UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
			MaxUploadSize: int64(getEnvAsInt("MAX_UPLOAD_SIZE", 50)) * 1024 * 1024, // MB to Bytes
			AllowedTypes:  []string{".pde", ".png", ".jpg", ".jpeg", ".gif", ".webp"},
		},
		AWS: AWSConfig{
			Region:         getEnv("AWS_REGION", "ap-northeast-1"),
			LambdaEndpoint: getEnv("AWS_LAMBDA_ENDPOINT", ""),
		},
		CloudflareWorker: CloudflareWorkerConfig{
			URL:     getEnv("CLOUDFLARE_WORKER_URL", ""),
			APIKey:  getEnv("CLOUDFLARE_API_KEY", ""),
			Bucket:  getEnv("CLOUDFLARE_BUCKET", "sketchshifter-bucket"),
			Enabled: getEnvAsBool("CLOUDFLARE_ENABLED", false),
		},
	}

	return config, nil
}

// getEnv 環境変数を取得、存在しない場合はデフォルト値を返す
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt 環境変数を整数として取得
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsBool 環境変数をboolとして取得
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
