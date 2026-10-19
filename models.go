package main

type Flat struct {
	ID      int64  `json:"id"`
	Address string `json:"address"`
	Price   int    `json:"price"`
	Square  int    `json:"square"`
}

type CreateFlatRequest struct {
	Address string `json:"address"`
	Price   int    `json:"price"`
	Square  int    `json:"square"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}
