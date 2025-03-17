package controllers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"github.com/SketchShifter/sketchshifter_backend/internal/mock"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
)

// Register は新しいユーザーを登録する
func Register(c *gin.Context) {
	// リクエストボディの構造体
	type RegisterRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name" binding:"required"`
		Nickname string `json:"nickname" binding:"required"`
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// モックユーザーを返す
	token, err := utils.GenerateJWT(mock.Users[0].ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user":  mock.Users[0],
		"token": token,
	})
}

// Login はユーザーをログインさせる
func Login(c *gin.Context) {
	// リクエストボディの構造体
	type LoginRequest struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// モックユーザーを返す
	token, err := utils.GenerateJWT(mock.Users[0].ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  mock.Users[0],
		"token": token,
	})
}

// GetCurrentUser は現在ログインしているユーザーの情報を返す
func GetCurrentUser(c *gin.Context) {
	// モックユーザーを返す
	userID, _ := c.Get("userID")
	
	// モックデータから適切なユーザーを探す
	var user mock.Users[0]
	for _, u := range mock.Users {
		if u.ID == userID {
			user = u
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
