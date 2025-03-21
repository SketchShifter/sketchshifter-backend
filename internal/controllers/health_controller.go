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
}

// Check ヘルスチェック
func (c *HealthController) Check(ctx *gin.Context) {
	uptime := time.Since(c.startTime).String()
	status := &HealthStatus{
		Status:    "ok",
		Uptime:    uptime,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	ctx.JSON(http.StatusOK, status)
}