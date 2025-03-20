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
	OAuth(provider, code string) (*models.User, string, error)
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

// OAuth OAuthログイン/登録
func (s *authService) OAuth(provider, code string) (*models.User, string, error) {
	// OAuthプロバイダからユーザー情報を取得（実際の実装はプロバイダに応じて異なる）
	userInfo, err := s.getOAuthUserInfo(provider, code)
	if err != nil {
		return nil, "", err
	}

	// 既存のアカウントを確認
	user, err := s.userRepo.FindByExternalAccount(provider, userInfo.ID)
	if err == nil && user != nil {
		// 既存ユーザーを見つけた場合はログイン処理
		token, err := s.generateToken(user.ID)
		if err != nil {
			return nil, "", err
		}
		return user, token, nil
	}

	// メールアドレスで既存ユーザーを確認
	if userInfo.Email != "" {
		existingUser, err := s.userRepo.FindByEmail(userInfo.Email)
		if err == nil && existingUser != nil {
			// 既存ユーザーに外部アカウントを関連付け
			externalAccount := &models.ExternalAccount{
				UserID:     existingUser.ID,
				Provider:   provider,
				ExternalID: userInfo.ID,
			}
			if err := s.userRepo.CreateExternalAccount(externalAccount); err != nil {
				return nil, "", err
			}

			token, err := s.generateToken(existingUser.ID)
			if err != nil {
				return nil, "", err
			}
			return existingUser, token, nil
		}
	}

	// 新しいユーザーを作成
	newUser := &models.User{
		Email:    userInfo.Email,
		Password: "", // OAuth認証ではパスワードは不要
		Name:     userInfo.Name,
		Nickname: userInfo.Nickname,
	}

	if err := s.userRepo.Create(newUser); err != nil {
		return nil, "", err
	}

	// 外部アカウントを関連付け
	externalAccount := &models.ExternalAccount{
		UserID:     newUser.ID,
		Provider:   provider,
		ExternalID: userInfo.ID,
	}
	if err := s.userRepo.CreateExternalAccount(externalAccount); err != nil {
		return nil, "", err
	}

	// JWTトークンを生成
	token, err := s.generateToken(newUser.ID)
	if err != nil {
		return nil, "", err
	}

	return newUser, token, nil
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

// OAuthUserInfo OAuth認証で取得するユーザー情報
type OAuthUserInfo struct {
	ID       string
	Email    string
	Name     string
	Nickname string
}

// getOAuthUserInfo OAuthプロバイダからユーザー情報を取得（実装例）
func (s *authService) getOAuthUserInfo(provider, code string) (*OAuthUserInfo, error) {
	// この部分は実際のOAuthプロバイダに応じて実装する必要があります
	// ここでは例としてダミー実装を返します

	// 実際にはここでOAuthプロバイダのトークンエンドポイントにリクエストを送り、
	// アクセストークンを取得し、そのアクセストークンを使ってユーザー情報を取得する処理を実装します

	// Googleの場合
	if provider == "google" {
		// TODO: Google OAuth実装
	}

	// GitHubの場合
	if provider == "github" {
		// TODO: GitHub OAuth実装
	}

	// ダミー実装
	return &OAuthUserInfo{
		ID:       "dummy_id",
		Email:    "dummy@example.com",
		Name:     "Dummy User",
		Nickname: "dummy",
	}, nil
}

// auth_service.go に追加する実装
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
