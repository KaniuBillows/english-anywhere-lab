package dto

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthTokensDTO struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type UserDTO struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
}

type AuthResponse struct {
	User   UserDTO       `json:"user"`
	Tokens AuthTokensDTO `json:"tokens"`
}
