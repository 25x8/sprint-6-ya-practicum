package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/25x8/sprint-6-ya-practicum/internal/gophermart/auth"
)

// ContextKey - тип для ключей контекста
type ContextKey string

// Константы для ключей контекста
const (
	UserIDKey ContextKey = "user_id"
)

// AuthMiddleware - middleware для аутентификации
type AuthMiddleware struct {
	auth *auth.Auth
}

// NewAuthMiddleware создает новый экземпляр AuthMiddleware
func NewAuthMiddleware(auth *auth.Auth) *AuthMiddleware {
	return &AuthMiddleware{auth: auth}
}

// Middleware возвращает middleware для аутентификации
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Проверяем формат токена
		headerParts := strings.Split(authHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		// Проверяем токен
		userID, err := m.auth.ValidateToken(headerParts[1])
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Добавляем ID пользователя в контекст
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID возвращает ID пользователя из контекста
func GetUserID(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDKey).(int)
	return userID, ok
}
