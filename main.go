package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Println("Ошибка создания пула:", err)
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		fmt.Println("Ошибка подклчения к PostgreSQL:", err)
		return
	}

	server := &Server{db: pool}

	http.HandleFunc("/register", server.registerHandler)
	http.HandleFunc("/login", server.loginHandler)
	http.HandleFunc("/flats/search", server.searchFlatsHandler)
	http.HandleFunc("/flats", authMiddleware(server.createFlatHandler))

	fmt.Println("FlatMate API запущен на порту 8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Ошибка HTTP-сервера:", err)
	}
}
