package auth

import (
	"context"
	"errors"
	"time"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/repository"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// Ошибки аутентификации
var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// Claims представляет данные JWT-токена
type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

// Auth предоставляет функции для аутентификации пользователей
type Auth struct {
	repo       repository.Repository
	signingKey []byte
	tokenTTL   time.Duration
}

// NewAuth создает новый экземпляр Auth
func NewAuth(repo repository.Repository, signingKey string, tokenTTL time.Duration) *Auth {
	return &Auth{
		repo:       repo,
		signingKey: []byte(signingKey),
		tokenTTL:   tokenTTL,
	}
}

// Register регистрирует нового пользователя
func (a *Auth) Register(ctx context.Context, login, password string) (string, error) {
	// Проверяем, существует ли пользователь
	user, err := a.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", err
	}
	if user != nil {
		return "", repository.ErrUserExists
	}

	// Хешируем пароль
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Создаем пользователя
	userID, err := a.repo.CreateUser(ctx, login, string(passwordHash))
	if err != nil {
		return "", err
	}

	// Генерируем токен
	token, err := a.generateToken(userID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Login аутентифицирует пользователя
func (a *Auth) Login(ctx context.Context, login, password string) (string, error) {
	// Получаем пользователя
	user, err := a.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", repository.ErrInvalidCredentials
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", repository.ErrInvalidCredentials
	}

	// Генерируем токен
	token, err := a.generateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateToken проверяет токен и возвращает ID пользователя
func (a *Auth) ValidateToken(tokenString string) (int, error) {
	// Парсим токен
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return a.signingKey, nil
	})
	if err != nil {
		return 0, ErrInvalidToken
	}

	// Проверяем валидность токена
	if !token.Valid {
		return 0, ErrInvalidToken
	}

	// Получаем claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return 0, ErrInvalidToken
	}

	// Проверяем срок действия токена
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return 0, ErrExpiredToken
	}

	return claims.UserID, nil
}

// generateToken генерирует JWT-токен
func (a *Auth) generateToken(userID int) (string, error) {
	// Создаем claims
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Создаем токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	tokenString, err := token.SignedString(a.signingKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
