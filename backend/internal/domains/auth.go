package domains

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}
