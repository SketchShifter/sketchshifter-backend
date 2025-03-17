package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"github.com/SketchShifter/sketchshifter_backend/internal/mock"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"strings"
)

// GetWorks は作品の一覧を取得する
func GetWorks(c *gin.Context) {
	// クエリパラメータの取得
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	tag := c.Query("tag")
	userIDStr := c.Query("user_id")
	sort := c.DefaultQuery("sort", "newest")
	
	// モックデータをフィルタリング
	filteredWorks := mock.Works
	
	// 検索フィルタリング (タイトル検索のみ実装)
	if search != "" {
		var filtered []models.Work
		for _, work := range filteredWorks {
			if strings.Contains(strings.ToLower(work.Title), strings.ToLower(search)) {
				filtered = append(filtered, work)
			}
		}
		filteredWorks = filtered
	}
	
	// タグでフィルタリング
	if tag != "" {
		var filtered []models.Work
		for _, work := range filteredWorks {
			for _, t := range work.Tags {
				if strings.ToLower(t.Name) == strings.ToLower(tag) {
					filtered = append(filtered, work)
					break
				}
			}
		}
		filteredWorks = filtered
	}
	
	// ユーザーIDでフィルタリング
	if userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			var filtered []models.Work
			for _, work := range filteredWorks {
				if work.UserID != nil && *work.UserID == uint(userID) {
					filtered = append(filtered, work)
				}
			}
			filteredWorks = filtered
		}
	}
	
	// ページネーション (簡易実装)
	start := (page - 1) * limit
	end := start + limit
	if start >= len(filteredWorks) {
		start = 0
		end = 0
	}
	if end > len(filteredWorks) {
		end = len(filteredWorks)
	}
	
	pagedWorks := []models.Work{}
	if start < end {
		pagedWorks = filteredWorks[start:end]
	}
	
	// ユーザー情報を付与
	works := []gin.H{}
	for _, work := range pagedWorks {
		workWithUser := gin.H{
			"id":             work.ID,
			"title":          work.Title,
			"description":    work.Description,
			"file_url":       work.FileURL,
			"thumbnail_url":  work.ThumbnailURL,
			"code_shared":    work.CodeShared,
			"views":          work.Views,
			"is_guest":       work.IsGuest,
			"guest_nickname": work.GuestNickname,
			"created_at":     work.CreatedAt,
			"updated_at":     work.UpdatedAt,
			"tags":           work.Tags,
			"likes_count":    countLikes(work.ID),
			"comments_count": countComments(work.ID),
		}
		
		if work.UserID != nil {
			for _, user := range mock.Users {
				if user.ID == *work.UserID {
					workWithUser["user"] = user
					break
				}
			}
		}
		
		if work.CodeShared {
			workWithUser["code_content"] = work.CodeContent
		}
		
		works = append(works, workWithUser)
	}

	// 結果を返す
	c.JSON(http.StatusOK, gin.H{
		"works": works,
		"total": len(filteredWorks),
		"pages": (len(filteredWorks) + limit - 1) / limit,
		"page":  page,
	})
}

// GetWork は指定されたIDの作品を取得する
func GetWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータから作品を検索
	var work models.Work
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			work = w
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// ユーザー情報を付与
	workWithUser := gin.H{
		"id":             work.ID,
		"title":          work.Title,
		"description":    work.Description,
		"file_url":       work.FileURL,
		"thumbnail_url":  work.ThumbnailURL,
		"code_shared":    work.CodeShared,
		"views":          work.Views,
		"is_guest":       work.IsGuest,
		"guest_nickname": work.GuestNickname,
		"created_at":     work.CreatedAt,
		"updated_at":     work.UpdatedAt,
		"tags":           work.Tags,
		"likes_count":    countLikes(work.ID),
		"comments_count": countComments(work.ID),
	}
	
	if work.UserID != nil {
		for _, user := range mock.Users {
			if user.ID == *work.UserID {
				workWithUser["user"] = user
				break
			}
		}
	}
	
	if work.CodeShared {
		workWithUser["code_content"] = work.CodeContent
	}
	
	c.JSON(http.StatusOK, workWithUser)
}

// CreateWork は新しい作品を作成する
func CreateWork(c *gin.Context) {
	// マルチパートフォームデータを受け取る想定
	title := c.PostForm("title")
	description := c.PostForm("description")
	codeSharedStr := c.DefaultPostForm("code_shared", "false")
	codeContent := c.PostForm("code_content")
	tagsList := c.PostForm("tags") // カンマ区切りを想定
	
	// バリデーション
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required"})
		return
	}
	
	// モックデータを作成
	userID, _ := c.Get("userID")
	uid := uint(userID.(uint))
	
	// タグ処理
	var tags []models.Tag
	if tagsList != "" {
		tagNames := strings.Split(tagsList, ",")
		for _, name := range tagNames {
			name = strings.TrimSpace(name)
			for _, tag := range mock.Tags {
				if strings.ToLower(tag.Name) == strings.ToLower(name) {
					tags = append(tags, tag)
					break
				}
			}
		}
	}
	
	// 新しい作品を作成
	newWork := models.Work{
		ID:           uint(len(mock.Works) + 1),
		UserID:       &uid,
		Title:        title,
		Description:  description,
		FileURL:      fmt.Sprintf("/uploads/works/work_%d.pde", len(mock.Works)+1),
		ThumbnailURL: "https://via.placeholder.com/300x200",
		CodeShared:   codeSharedStr == "true",
		CodeContent:  codeContent,
		Views:        0,
		IsGuest:      false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Tags:         tags,
	}
	
	// モックデータのレスポンス
	workWithUser := gin.H{
		"id":             newWork.ID,
		"title":          newWork.Title,
		"description":    newWork.Description,
		"file_url":       newWork.FileURL,
		"thumbnail_url":  newWork.ThumbnailURL,
		"code_shared":    newWork.CodeShared,
		"views":          newWork.Views,
		"is_guest":       newWork.IsGuest,
		"created_at":     newWork.CreatedAt,
		"updated_at":     newWork.UpdatedAt,
		"tags":           newWork.Tags,
		"likes_count":    0,
		"comments_count": 0,
	}
	
	for _, user := range mock.Users {
		if user.ID == uid {
			workWithUser["user"] = user
			break
		}
	}
	
	if newWork.CodeShared {
		workWithUser["code_content"] = newWork.CodeContent
	}
	
	c.JSON(http.StatusCreated, workWithUser)
}

// UpdateWork は指定されたIDの作品を更新する
func UpdateWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// マルチパートフォームデータを受け取る想定
	title := c.PostForm("title")
	description := c.PostForm("description")
	codeSharedStr := c.PostForm("code_shared")
	codeContent := c.PostForm("code_content")
	tagsList := c.PostForm("tags") // カンマ区切りを想定
	
	// モックデータから作品を検索
	var work models.Work
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			work = w
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// 権限チェック
	userID, _ := c.Get("userID")
	if work.UserID == nil || *work.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this work"})
		return
	}
	
	// 更新
	if title != "" {
		work.Title = title
	}
	if description != "" {
		work.Description = description
	}
	if codeSharedStr != "" {
		work.CodeShared = codeSharedStr == "true"
	}
	if codeContent != "" && work.CodeShared {
		work.CodeContent = codeContent
	}
	
	// タグ処理
	if tagsList != "" {
		var tags []models.Tag
		tagNames := strings.Split(tagsList, ",")
		for _, name := range tagNames {
			name = strings.TrimSpace(name)
			for _, tag := range mock.Tags {
				if strings.ToLower(tag.Name) == strings.ToLower(name) {
					tags = append(tags, tag)
					break
				}
			}
		}
		work.Tags = tags
	}
	
	work.UpdatedAt = time.Now()
	
	// レスポンス
	workWithUser := gin.H{
		"id":             work.ID,
		"title":          work.Title,
		"description":    work.Description,
		"file_url":       work.FileURL,
		"thumbnail_url":  work.ThumbnailURL,
		"code_shared":    work.CodeShared,
		"views":          work.Views,
		"is_guest":       work.IsGuest,
		"created_at":     work.CreatedAt,
		"updated_at":     work.UpdatedAt,
		"tags":           work.Tags,
		"likes_count":    countLikes(work.ID),
		"comments_count": countComments(work.ID),
	}
	
	if work.UserID != nil {
		for _, user := range mock.Users {
			if user.ID == *work.UserID {
				workWithUser["user"] = user
				break
			}
		}
	}
	
	if work.CodeShared {
		workWithUser["code_content"] = work.CodeContent
	}
	
	c.JSON(http.StatusOK, workWithUser)
}

cat >> internal/controllers/works.go << 'EOF'
CodeContent
	}
	
	c.JSON(http.StatusOK, workWithUser)
}

// DeleteWork は指定されたIDの作品を削除する
func DeleteWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータから作品を検索
	var work models.Work
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			work = w
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// 権限チェック
	userID, _ := c.Get("userID")
	if work.UserID == nil || *work.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this work"})
		return
	}
	
	// 削除完了レスポンス
	c.Status(http.StatusNoContent)
}

// PreviewWork は作品をプレビューする
func PreviewWork(c *gin.Context) {
	// マルチパートフォームデータのファイルを受け取る想定
	file, err := c.FormFile("file")
	code := c.PostForm("code")
	
	if err != nil && code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File or code is required"})
		return
	}
	
	// モックレスポンス
	c.JSON(http.StatusOK, gin.H{
		"preview_url": "https://preview.example.com/temp/preview_12345.html",
	})
}

// LikeWork は作品にいいねを追加する
func LikeWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータから作品を検索
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// いいね追加
	userID, _ := c.Get("userID")
	uid := userID.(uint)
	
	// 現在のいいね数を取得
	count := countLikes(uint(id))
	
	// レスポンス
	c.JSON(http.StatusOK, gin.H{
		"likes_count": count + 1,
	})
}

// UnlikeWork は作品のいいねを解除する
func UnlikeWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータから作品を検索
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// いいね解除
	userID, _ := c.Get("userID")
	uid := userID.(uint)
	
	// 現在のいいね数を取得
	count := countLikes(uint(id))
	newCount := count
	if newCount > 0 {
		newCount--
	}
	
	// レスポンス
	c.JSON(http.StatusOK, gin.H{
		"likes_count": newCount,
	})
}

// FavoriteWork は作品をお気に入りに追加する
func FavoriteWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータから作品を検索
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// お気に入り追加（モック）
	c.Status(http.StatusOK)
}

// UnfavoriteWork は作品のお気に入りを解除する
func UnfavoriteWork(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータから作品を検索
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// お気に入り解除（モック）
	c.Status(http.StatusOK)
}

// GetWorkComments は作品のコメント一覧を取得する
func GetWorkComments(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// ページネーションパラメータ
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	// モックデータから作品を検索
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// コメントを取得
	var comments []models.Comment
	for _, comment := range mock.Comments {
		if comment.WorkID == uint(id) {
			comments = append(comments, comment)
		}
	}
	
	// ページネーション
	start := (page - 1) * limit
	end := start + limit
	if start >= len(comments) {
		start = 0
		end = 0
	}
	if end > len(comments) {
		end = len(comments)
	}
	
	pagedComments := []models.Comment{}
	if start < end {
		pagedComments = comments[start:end]
	}
	
	// ユーザー情報を付与
	commentsWithUser := []gin.H{}
	for _, comment := range pagedComments {
		commentWithUser := gin.H{
			"id":             comment.ID,
			"content":        comment.Content,
			"is_guest":       comment.IsGuest,
			"guest_nickname": comment.GuestNickname,
			"created_at":     comment.CreatedAt,
			"updated_at":     comment.UpdatedAt,
		}
		
		if comment.UserID != nil {
			for _, user := range mock.Users {
				if user.ID == *comment.UserID {
					commentWithUser["user"] = user
					break
				}
			}
		}
		
		commentsWithUser = append(commentsWithUser, commentWithUser)
	}
	
	// レスポンス
	c.JSON(http.StatusOK, gin.H{
		"comments": commentsWithUser,
		"total":    len(comments),
		"pages":    (len(comments) + limit - 1) / limit,
		"page":     page,
	})
}

// CreateComment は作品にコメントを追加する
func CreateComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// リクエストボディ
	var req struct {
		Content       string `json:"content" binding:"required"`
		IsGuest       bool   `json:"is_guest"`
		GuestNickname string `json:"guest_nickname"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// モックデータから作品を検索
	found := false
	for _, w := range mock.Works {
		if w.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Work not found"})
		return
	}
	
	// 新しいコメントを作成
	var userID *uint
	if !req.IsGuest {
		uid, exists := c.Get("userID")
		if exists {
			id := uid.(uint)
			userID = &id
		}
	}
	
	newComment := models.Comment{
		ID:            uint(len(mock.Comments) + 1),
		WorkID:        uint(id),
		UserID:        userID,
		Content:       req.Content,
		IsGuest:       req.IsGuest,
		GuestNickname: req.GuestNickname,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	// レスポンス作成
	commentWithUser := gin.H{
		"id":             newComment.ID,
		"content":        newComment.Content,
		"is_guest":       newComment.IsGuest,
		"guest_nickname": newComment.GuestNickname,
		"created_at":     newComment.CreatedAt,
		"updated_at":     newComment.UpdatedAt,
	}
	
	if newComment.UserID != nil {
		for _, user := range mock.Users {
			if user.ID == *newComment.UserID {
				commentWithUser["user"] = user
				break
			}
		}
	}
	
	c.JSON(http.StatusCreated, commentWithUser)
}

// UpdateComment はコメントを更新する
func UpdateComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// リクエストボディ
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// モックデータからコメントを検索
	var comment models.Comment
	found := false
	for _, cmnt := range mock.Comments {
		if cmnt.ID == uint(id) {
			comment = cmnt
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	
	// 権限チェック
	userID, _ := c.Get("userID")
	if comment.UserID == nil || *comment.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this comment"})
		return
	}
	
	// 更新
	comment.Content = req.Content
	comment.UpdatedAt = time.Now()
	
	// レスポンス
	commentWithUser := gin.H{
		"id":             comment.ID,
		"content":        comment.Content,
		"is_guest":       comment.IsGuest,
		"guest_nickname": comment.GuestNickname,
		"created_at":     comment.CreatedAt,
		"updated_at":     comment.UpdatedAt,
	}
	
	if comment.UserID != nil {
		for _, user := range mock.Users {
			if user.ID == *comment.UserID {
				commentWithUser["user"] = user
				break
			}
		}
	}
	
	c.JSON(http.StatusOK, commentWithUser)
}

// DeleteComment はコメントを削除する
func DeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータからコメントを検索
	var comment models.Comment
	found := false
	for _, cmnt := range mock.Comments {
		if cmnt.ID == uint(id) {
			comment = cmnt
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	
	// 権限チェック
	userID, _ := c.Get("userID")
	if comment.UserID == nil || *comment.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this comment"})
		return
	}
	
	// 削除完了レスポンス
	c.Status(http.StatusNoContent)
}

// GetTags はタグ一覧を取得する
func GetTags(c *gin.Context) {
	// クエリパラメータ
	search := c.Query("search")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	
	// モックデータをフィルタリング
	filteredTags := mock.Tags
	
	// 検索フィルタリング
	if search != "" {
		var filtered []models.Tag
		for _, tag := range filteredTags {
			if strings.Contains(strings.ToLower(tag.Name), strings.ToLower(search)) {
				filtered = append(filtered, tag)
			}
		}
		filteredTags = filtered
	}
	
	// 結果の制限
	if len(filteredTags) > limit {
		filteredTags = filteredTags[:limit]
	}
	
	c.JSON(http.StatusOK, filteredTags)
}

// GetUserFavorites はユーザーのお気に入り作品一覧を取得する
func GetUserFavorites(c *gin.Context) {
	// ページネーションパラメータ
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	// 現在のユーザーID
	userID, _ := c.Get("userID")
	uid := userID.(uint)
	
	// モックデータからお気に入りを検索
	var favorites []models.Favorite
	for _, fav := range mock.Favorites {
		if fav.UserID == uid {
			favorites = append(favorites, fav)
		}
	}
	
	// お気に入りの作品IDを取得
	var favoriteWorkIDs []uint
	for _, fav := range favorites {
		favoriteWorkIDs = append(favoriteWorkIDs, fav.WorkID)
	}
	
	// 作品を取得
	var works []models.Work
	for _, work := range mock.Works {
		for _, favID := range favoriteWorkIDs {
			if work.ID == favID {
				works = append(works, work)
				break
			}
		}
	}
	
	// ページネーション
	start := (page - 1) * limit
	end := start + limit
	if start >= len(works) {
		start = 0
		end = 0
	}
	if end > len(works) {
		end = len(works)
	}
	
	pagedWorks := []models.Work{}
	if start < end {
		pagedWorks = works[start:end]
	}
	
	// ユーザー情報を付与
	worksWithUser := []gin.H{}
	for _, work := range pagedWorks {
		workWithUser := gin.H{
			"id":             work.ID,
			"title":          work.Title,
			"description":    work.Description,
			"file_url":       work.FileURL,
			"thumbnail_url":  work.ThumbnailURL,
			"code_shared":    work.CodeShared,
			"views":          work.Views,
			"is_guest":       work.IsGuest,
			"guest_nickname": work.GuestNickname,
			"created_at":     work.CreatedAt,
			"updated_at":     work.UpdatedAt,
			"tags":           work.Tags,
			"likes_count":    countLikes(work.ID),
			"comments_count": countComments(work.ID),
		}
		
		if work.UserID != nil {
			for _, user := range mock.Users {
				if user.ID == *work.UserID {
					workWithUser["user"] = user
					break
				}
			}
		}
		
		if work.CodeShared {
			workWithUser["code_content"] = work.CodeContent
		}
		
		worksWithUser = append(worksWithUser, workWithUser)
	}
	
	// レスポンス
	c.JSON(http.StatusOK, gin.H{
		"works": worksWithUser,
		"total": len(works),
		"pages": (len(works) + limit - 1) / limit,
		"page":  page,
	})
}

// GetUser はユーザー情報を取得する
func GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// モックデータからユーザーを検索
	var user models.User
	found := false
	for _, u := range mock.Users {
		if u.ID == uint(id) {
			user = u
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	c.JSON(http.StatusOK, user)
}

// GetUserWorks はユーザーの作品一覧を取得する
func GetUserWorks(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	
	// ページネーションパラメータ
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	// モックデータからユーザーを検索
	found := false
	for _, u := range mock.Users {
		if u.ID == uint(id) {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	// ユーザーの作品を取得
	var works []models.Work
	for _, work := range mock.Works {
		if work.UserID != nil && *work.UserID == uint(id) {
			works = append(works, work)
		}
	}
	
	// ページネーション
	start := (page - 1) * limit
	end := start + limit
	if start >= len(works) {
		start = 0
		end = 0
	}
	if end > len(works) {
		end = len(works)
	}
	
	pagedWorks := []models.Work{}
	if start < end {
		pagedWorks = works[start:end]
	}
	
	// ユーザー情報を付与
	worksWithUser := []gin.H{}
	for _, work := range pagedWorks {
		workWithUser := gin.H{
			"id":             work.ID,
			"title":          work.Title,
			"description":    work.Description,
			"file_url":       work.FileURL,
			"thumbnail_url":  work.ThumbnailURL,
			"code_shared":    work.CodeShared,
			"views":          work.Views,
			"is_guest":       work.IsGuest,
			"guest_nickname": work.GuestNickname,
			"created_at":     work.CreatedAt,
			"updated_at":     work.UpdatedAt,
			"tags":           work.Tags,
			"likes_count":    countLikes(work.ID),
			"comments_count": countComments(work.ID),
		}
		
		if work.UserID != nil {
			for _, user := range mock.Users {
				if user.ID == *work.UserID {
					workWithUser["user"] = user
					break
				}
			}
		}
		
		if work.CodeShared {
			workWithUser["code_content"] = work.CodeContent
		}
		
		worksWithUser = append(worksWithUser, workWithUser)
	}
	
	// レスポンス
	c.JSON(http.StatusOK, gin.H{
		"works": worksWithUser,
		"total": len(works),
		"pages": (len(works) + limit - 1) / limit,
		"page":  page,
	})
}

// ヘルパー関数

// countLikes は作品のいいね数を数える
func countLikes(workID uint) int {
	count := 0
	for _, like := range mock.Likes {
		if like.WorkID == workID {
			count++
		}
	}
	return count
}

// countComments は作品のコメント数を数える
func countComments(workID uint) int {
	count := 0
	for _, comment := range mock.Comments {
		if comment.WorkID == workID {
			count++
		}
	}
	return count
}
