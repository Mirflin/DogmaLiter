package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextUserID   contextKey = "user_id"
	ContextUsername contextKey = "username"
	ContextUserRole contextKey = "user_role"
)

func JWTMiddleware(jwtManager *JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondJSON(w, 401, map[string]string{"error": "Required authentication"})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondJSON(w, 401, map[string]string{"error": "Invalid token format"})
				return
			}

			claims, err := jwtManager.ValidateToken(parts[1])
			if err != nil {
				respondJSON(w, 401, map[string]string{"error": "Invalid or expired token"})
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextUsername, claims.Username)
			ctx = context.WithValue(ctx, ContextUserRole, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(ContextUserRole).(string)
		if !ok || role != "admin" {
			respondJSON(w, 403, map[string]string{"error": "Access restricted to administrators only"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(r *http.Request) string {
	val, _ := r.Context().Value(ContextUserID).(string)
	return val
}

func GetUserRole(r *http.Request) string {
	val, _ := r.Context().Value(ContextUserRole).(string)
	return val
}
