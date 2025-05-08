package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
)

// CommentController コメントに関するコントローラー
type CommentController struct {
	commentService services.CommentService
}

// NewCommentController CommentControllerを作成
func NewCommentController(commentService services.CommentService) *CommentController {
	return &CommentController{
		commentService: commentService,
	}
}

// CommentRequest コメントリクエスト
type CommentRequest struct {
	Content       string `json:"content" binding:"required"`
	IsGuest       bool   `json:"is_guest"`
	GuestNickname string `json:"guest_nickname"`
}

// Create 新しいコメントを作成
func (c *CommentController) Create(ctx *gin.Context) {
	// IDを解析
	workID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	var req CommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ユーザー情報を取得（非ゲストコメントの場合）
	var userID *uint
	if !req.IsGuest {
		user, exists := ctx.Get("user")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
			return
		}
		u := user.(*models.User)
		userID = &u.ID
	}

	// コメントを作成
	comment, err := c.commentService.Create(
		req.Content,
		uint(workID),
		userID,
		req.IsGuest,
		req.GuestNickname,
	)
	if err != nil {
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"comment": comment})
}

// Update コメントを更新
func (c *CommentController) Update(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	var req CommentRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// コメントを更新
	comment, err := c.commentService.Update(uint(id), u.ID, req.Content)
	if err != nil {
		if strings.Contains(err.Error(), "権限がありません") {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"comment": comment})
}

// Delete コメントを削除
func (c *CommentController) Delete(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// コメントを削除
	if err := c.commentService.Delete(uint(id), u.ID); err != nil {
		if strings.Contains(err.Error(), "権限がありません") {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// List 作品のコメント一覧を取得
func (c *CommentController) List(ctx *gin.Context) {
	// 作品IDを解析
	workID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// クエリパラメータを取得
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")

	// 数値パラメータを解析
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// コメント一覧を取得
	comments, total, pages, err := c.commentService.ListByWork(uint(workID), page, limit)
	if err != nil {
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"comments": comments,
		"total":    total,
		"pages":    pages,
		"page":     page,
	})
}
