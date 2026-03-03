package handler

import (
	"errors"
	"net/http"

	"github.com/bennyshi/english-anywhere-lab/internal/auth"
	"github.com/bennyshi/english-anywhere-lab/internal/transport/http/dto"
)

type AuthHandler struct {
	svc *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	result, err := h.svc.Register(r.Context(), auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Locale:   req.Locale,
		Timezone: req.Timezone,
	})
	if errors.Is(err, auth.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "CONFLICT", "email already registered")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "registration failed")
		return
	}

	writeJSON(w, http.StatusCreated, dto.AuthResponse{
		User: dto.UserDTO{
			ID:       result.User.ID,
			Email:    result.User.Email,
			Locale:   result.User.Locale,
			Timezone: result.User.Timezone,
		},
		Tokens: dto.AuthTokensDTO{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresIn:    result.Tokens.ExpiresIn,
		},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	result, err := h.svc.Login(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if errors.Is(err, auth.ErrInvalidCreds) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "login failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.AuthResponse{
		User: dto.UserDTO{
			ID:       result.User.ID,
			Email:    result.User.Email,
			Locale:   result.User.Locale,
			Timezone: result.User.Timezone,
		},
		Tokens: dto.AuthTokensDTO{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			ExpiresIn:    result.Tokens.ExpiresIn,
		},
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeAndValidate(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	tokens, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "refresh failed")
		return
	}

	writeJSON(w, http.StatusOK, dto.AuthTokensDTO{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}
