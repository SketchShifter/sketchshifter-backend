package controllers

import (
	"net/http"
	"strconv"

	"github.com/SketchShifter/sketchshifter_backend/internal/services"
	"github.com/gin-gonic/gin"
)

// TagController タグに関するコントローラー
type TagController struct {
	tagService services.TagService
}

// NewTagController TagControllerを作成
func NewTagController(tagService services.TagService) *TagController {
	return &TagController{
		tagService: tagService,
	}
}

// List タグ一覧を取得
func (c *TagController) List(ctx *gin.Context) {
	// クエリパラメータを取得
	search := ctx.Query("search")
	limitStr := ctx.DefaultQuery("limit", "50")

	// リミットを解析
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 200 {
		limit = 50
	}

	// タグ一覧を取得
	tags, err := c.tagService.List(search, limit)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, tags)
}
