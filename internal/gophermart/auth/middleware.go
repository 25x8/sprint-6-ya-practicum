package auth

import (
	"context"
	"net/http"
)

// Константа для ключа userID в контексте
const userIDKey = "userID"

// AuthMiddleware создает middleware для проверки JWT токена
func (a *Auth) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем cookie с токеном
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Извлекаем токен из cookie
		tokenString := cookie.Value

		// Проверяем токен и получаем ID пользователя
		userID, err := a.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Устанавливаем ID пользователя в контекст
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext извлекает ID пользователя из контекста
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDKey).(int)
	return userID, ok
}
