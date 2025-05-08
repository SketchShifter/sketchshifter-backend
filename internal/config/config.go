package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config アプリケーション設定に追加
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Auth       AuthConfig
	Lambda     LambdaConfig
	Cloudinary CloudinaryConfig // 追加
}

// CloudinaryConfig Cloudinary設定
type CloudinaryConfig struct {
	CloudName string
	APIKey    string
	APISecret string
	Folder    string
}

// ServerConfig サーバー設定
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	APIBaseURL   string
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

// LambdaConfig Lambda設定
type LambdaConfig struct {
	Region        string
	FunctionName  string
	RoleARN       string
	VpcID         string
	SubnetIDs     []string
	SecurityGroup string
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
			APIBaseURL:   getEnv("API_BASE_URL", "http://localhost:8080"),
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
		Lambda: LambdaConfig{
			Region:        getEnv("AWS_REGION", "ap-northeast-1"),
			FunctionName:  getEnv("AWS_LAMBDA_FUNCTION", "pde-converter"),
			RoleARN:       getEnv("AWS_LAMBDA_ROLE", ""),
			VpcID:         getEnv("AWS_VPC_ID", ""),
			SubnetIDs:     getEnvAsStringSlice("AWS_SUBNET_IDS", ",", []string{}),
			SecurityGroup: getEnv("AWS_SECURITY_GROUP", ""),
		},
		Cloudinary: CloudinaryConfig{
			CloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
			APIKey:    getEnv("CLOUDINARY_API_KEY", ""),
			APISecret: getEnv("CLOUDINARY_API_SECRET", ""),
			Folder:    getEnv("CLOUDINARY_FOLDER", "sketchshifter"),
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

// getEnvAsStringSlice 環境変数を文字列スライスとして取得
func getEnvAsStringSlice(key string, sep string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	values := []string{}
	for _, item := range strings.Split(valueStr, sep) {
		if item != "" {
			values = append(values, strings.TrimSpace(item))
		}
	}
	return values
}
