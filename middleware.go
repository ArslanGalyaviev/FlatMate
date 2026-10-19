package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		secret := os.Getenv("JWT_SECRET")
		if secret == "" {
			http.Error(w, "Ошибка конфигурации сервера", http.StatusInternalServerError)
			return
		}

		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {
				if token.Method != jwt.SigningMethodHS256 {
					return nil, fmt.Errorf("неожиданный алгоритм подписи")
				}

				return []byte(secret), nil
			},
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithExpirationRequired(),
		)

		if err != nil || !token.Valid {
			http.Error(w, "Недействительный токен", http.StatusUnauthorized)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Некорректные данные токена", http.StatusUnauthorized)
			return
		}

		role, ok := claims["role"].(string)
		if !ok {
			http.Error(w, "Роль не указана", http.StatusUnauthorized)
			return
		}

		if role != "owner" {
			http.Error(w, "Доступ разрешён только собственникам", http.StatusForbidden)
			return
		}
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok || userIDFloat <= 0 {
			http.Error(w, "Некорректный ID пользователя", http.StatusUnauthorized)
			return
		}

		userID := int64(userIDFloat)

		ctx := context.WithValue(r.Context(), userIDKey, userID)

		next(w, r.WithContext(ctx))
	}
}
