package services

import (
	"errors"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 認証に関するサービスインターフェース
type AuthService interface {
	Register(email, password, name, nickname string) (*models.User, string, error)
	Login(email, password string) (*models.User, string, error)
	ValidateToken(tokenString string) (*Claims, error)
	GetUserFromToken(tokenString string) (*models.User, error)
	ChangePassword(userID uint, currentPassword, newPassword string) error
}

// authService AuthServiceの実装
type authService struct {
	userRepo repository.UserRepository
	config   *config.Config
}

// NewAuthService AuthServiceを作成
func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) AuthService {
	return &authService{
		userRepo: userRepo,
		config:   cfg,
	}
}

// Claims JWTのペイロード
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// Register ユーザー登録
func (s *authService) Register(email, password, name, nickname string) (*models.User, string, error) {
	// メールアドレスが既に使用されているか確認
	existingUser, err := s.userRepo.FindByEmail(email)
	if err == nil && existingUser != nil {
		return nil, "", errors.New("このメールアドレスは既に使用されています")
	}

	// パスワードをハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// 新しいユーザーを作成
	user := &models.User{
		Email:    email,
		Password: string(hashedPassword),
		Name:     name,
		Nickname: nickname,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", err
	}

	// JWTトークンを生成
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// Login ログイン
func (s *authService) Login(email, password string) (*models.User, string, error) {
	// ユーザーを検索
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, "", errors.New("メールアドレスまたはパスワードが正しくありません")
	}

	// パスワードを検証
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("メールアドレスまたはパスワードが正しくありません")
	}

	// JWTトークンを生成
	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// ValidateToken トークンを検証
func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	// トークンを解析
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.Auth.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("無効なトークンです")
	}

	return claims, nil
}

// GetUserFromToken トークンからユーザーを取得
func (s *authService) GetUserFromToken(tokenString string) (*models.User, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword ユーザーのパスワードを変更
func (s *authService) ChangePassword(userID uint, currentPassword, newPassword string) error {
	// ユーザーを取得
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	// 現在のパスワードを検証
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
		return errors.New("現在のパスワードが正しくありません")
	}

	// 新しいパスワードをハッシュ化
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// パスワードを更新
	user.Password = string(hashedPassword)
	return s.userRepo.Update(user)
}

// generateToken JWTトークンを生成
func (s *authService) generateToken(userID uint) (string, error) {
	// トークンの有効期限を設定
	expirationTime := time.Now().Add(s.config.Auth.TokenExpiry)

	// クレームを作成
	claims := &Claims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// トークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Auth.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
