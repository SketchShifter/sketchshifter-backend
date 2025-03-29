package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/SketchShifter/sketchshifter_backend/internal/services"
	"github.com/gin-gonic/gin"
)

// LambdaController Lambda関数の操作を処理するコントローラー
type LambdaController struct {
	lambdaService services.LambdaService
}

// NewLambdaController LambdaControllerを作成
func NewLambdaController(lambdaService services.LambdaService) *LambdaController {
	return &LambdaController{
		lambdaService: lambdaService,
	}
}

// ProcessPDE 特定のProcessingIDのPDE変換を実行
func (c *LambdaController) ProcessPDE(ctx *gin.Context) {
	// ProcessingIDを取得
	processingIDStr := ctx.Param("id")
	if processingIDStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "ProcessingIDが必要です",
		})
		return
	}

	// ProcessingIDを数値に変換
	processingID, err := strconv.ParseUint(processingIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("無効なProcessingID: %v", err),
		})
		return
	}

	// PDE変換を実行
	err = c.lambdaService.InvokePDEConversion(uint(processingID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("PDE変換処理に失敗しました: %v", err),
		})
		return
	}

	// 成功レスポンス
	ctx.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "PDE変換処理が完了しました",
		"processingId": processingID,
	})
}
