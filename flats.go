package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

func (s *Server) searchFlatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		http.Error(w, "Некорректный user_id", http.StatusBadRequest)
		return
	}

	var exists bool
	err = s.db.QueryRow(
		r.Context(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)",
		userID,
	).Scan(&exists)

	if err != nil {
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	rows, err := s.db.Query(
		r.Context(),
		`SELECT f.id, f.address, f.price, f.square
		FROM flats AS f
		JOIN users AS u ON u.id = $1
		WHERE f.price <= u.max_price
		AND f.square >= u.min_square`,
		userID,
	)

	if err != nil {
		fmt.Println("Ошибка поиска квартир:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	flats := []Flat{}

	for rows.Next() {
		var flat Flat

		err := rows.Scan(
			&flat.ID,
			&flat.Address,
			&flat.Price,
			&flat.Square,
		)
		if err != nil {
			http.Error(w, "Ошибка чтения квартиры", http.StatusInternalServerError)
			return
		}

		flats = append(flats, flat)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Ошибка получения квартир", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(flats)
	if err != nil {
		fmt.Println("Ошибка отправки JSON:", err)
	}
}

func (s *Server) createFlatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}
	ownerID, ok := r.Context().Value(userIDKey).(int64)
	if !ok || ownerID <= 0 {
		http.Error(w, "Не удалось определить владельца", http.StatusUnauthorized)
		return
	}

	var req CreateFlatRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Некорректный JSON", http.StatusBadRequest)
		return
	}

	if req.Address == "" || req.Price <= 0 || req.Square <= 0 {
		http.Error(w, "Некорректные данные квартиры", http.StatusBadRequest)
		return
	}

	var flatID int64

	err = s.db.QueryRow(
		r.Context(),
		`INSERT INTO flats (address, price, square, owner_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		req.Address,
		req.Price,
		req.Square,
		ownerID,
	).Scan(&flatID)

	if err != nil {
		fmt.Println("Ошибка сохранения квартиры:", err)
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Квартира создана с ID: %d\n", flatID)
}
