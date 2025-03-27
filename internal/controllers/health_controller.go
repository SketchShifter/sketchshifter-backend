package controllers

import (
	"net/http"

	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
)

// HealthController ヘルスチェックに関するコントローラー
type HealthController struct {
	healthService services.HealthService
}

// NewHealthController HealthControllerを作成
func NewHealthController(healthService services.HealthService) *HealthController {
	return &HealthController{
		healthService: healthService,
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
	status, uptime, timestamp := c.healthService.GetStatus()

	healthStatus := &HealthStatus{
		Status:    status,
		Uptime:    uptime,
		Timestamp: timestamp,
		Version:   "1.0.0", // アプリケーションバージョン
	}

	ctx.JSON(http.StatusOK, healthStatus)
}
