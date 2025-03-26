package config

// CloudflareConfig Cloudflare設定
type CloudflareConfig struct {
	WorkerURL    string
	APIKey       string
	AccountID    string
	APIToken     string
	R2BucketName string
}

// GetCloudflareConfig Cloudflare設定を取得
func GetCloudflareConfig() *CloudflareConfig {
	return &CloudflareConfig{
		WorkerURL:    getEnv("CLOUDFLARE_WORKER_URL", ""),
		AccountID:    getEnv("CLOUDFLARE_ACCOUNT_ID", ""),
		APIToken:     getEnv("CLOUDFLARE_API_TOKEN", ""),
		R2BucketName: getEnv("R2_BUCKET_NAME", "sketchshifter-uploads"),
	}
}
