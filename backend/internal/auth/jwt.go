package auth

import (
	"fmt"
	"time"

	"github.com/bennyshi/english-anywhere-lab/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	issuer     string
	signKey    []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
}

type Claims struct {
	jwt.RegisteredClaims
	TokenType string `json:"type"`
}

func NewJWTManager(cfg *config.Config) *JWTManager {
	return &JWTManager{
		issuer:     cfg.JWTIssuer,
		signKey:    []byte(cfg.JWTSignKey),
		accessTTL:  cfg.JWTAccessTTL,
		refreshTTL: cfg.JWTRefreshTTL,
	}
}

func (j *JWTManager) GenerateTokenPair(userID string) (*TokenPair, error) {
	now := time.Now()

	accessClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTTL)),
		},
		TokenType: "access",
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(j.signKey)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTTL)),
		},
		TokenType: "refresh",
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(j.signKey)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(j.accessTTL.Seconds()),
	}, nil
}

func (j *JWTManager) ParseAccessToken(tokenStr string) (string, error) {
	return j.parseToken(tokenStr, "access")
}

func (j *JWTManager) ParseRefreshToken(tokenStr string) (string, error) {
	return j.parseToken(tokenStr, "refresh")
}

func (j *JWTManager) parseToken(tokenStr, expectedType string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return j.signKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	if claims.TokenType != expectedType {
		return "", fmt.Errorf("expected %s token, got %s", expectedType, claims.TokenType)
	}

	return claims.Subject, nil
}
