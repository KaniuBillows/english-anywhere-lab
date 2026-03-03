package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/bennyshi/english-anywhere-lab/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func Auth(jwtMgr *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			userID, err := jwtMgr.ParseAccessToken(parts[1])
			if err != nil {
				http.Error(w, `{"code":"UNAUTHORIZED","message":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}
