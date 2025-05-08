package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthController ヘルスチェックに関するコントローラー
type HealthController struct {
	startTime time.Time
}

// NewHealthController HealthControllerを作成
func NewHealthController() *HealthController {
	return &HealthController{
		startTime: time.Now(),
	}
}

// HealthStatus ヘルスステータスレスポンス
type HealthStatus struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Check ヘルスチェック
func (c *HealthController) Check(ctx *gin.Context) {
	status := "ok"
	uptime := time.Since(c.startTime).String()
	timestamp := time.Now().Format(time.RFC3339)

	healthStatus := &HealthStatus{
		Status:    status,
		Uptime:    uptime,
		Timestamp: timestamp,
		Version:   "1.0.0", // アプリケーションバージョン
	}

	ctx.JSON(http.StatusOK, healthStatus)
}
