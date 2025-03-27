package services

import (
	"time"
)

// HealthService ヘルスチェックに関するサービスインターフェース
type HealthService interface {
	GetStatus() (string, string, string)
}

// healthService HealthServiceの実装
type healthService struct {
	startTime time.Time
}

// NewHealthService HealthServiceを作成
func NewHealthService() HealthService {
	return &healthService{
		startTime: time.Now(),
	}
}

// GetStatus サービスのステータスを取得
func (s *healthService) GetStatus() (string, string, string) {
	uptime := time.Since(s.startTime).String()
	status := "ok"
	timestamp := time.Now().Format(time.RFC3339)

	return status, uptime, timestamp
}
