package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Заполните все поля", http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "tenant"
	}
	if req.Role != "tenant" && req.Role != "owner" {
		http.Error(w, "Недопустимая роль", http.StatusBadRequest)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		http.Error(w, "Ошибка обработки пароля", http.StatusInternalServerError)
		return
	}

	_, err = s.db.Exec(
		r.Context(),
		`INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4)`,
		req.Username,
		req.Email,
		string(passwordHash),
		req.Role,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Email уже зарегистрирован", http.StatusConflict)
			return
		}

		fmt.Println("Ошибка сохранения пользователя:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "Пользователь зарегистрирован!")
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Укажите email и пароль", http.StatusBadRequest)
		return
	}

	var user LoginResponse
	var passwordHash string

	err := s.db.QueryRow(
		r.Context(),
		`SELECT id, username, role, password_hash
		 FROM users
		 WHERE email = $1`,
		req.Email,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Role,
		&passwordHash,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
			return
		}

		fmt.Println("Ошибка поиска пользователя:", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(passwordHash),
		[]byte(req.Password),
	)

	if err != nil {
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		http.Error(w, "Ошибка конфигурации сервера", http.StatusInternalServerError)
		return
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		http.Error(w, "Ошибка создания токена", http.StatusInternalServerError)
		return
	}

	user.Token = tokenString
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(user); err != nil {
		fmt.Println("Ошибка отправки JSON:", err)
	}
}
